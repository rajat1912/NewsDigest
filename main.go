package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"news-agent/internal/config"
	"news-agent/internal/news"
	"news-agent/internal/storage"
	"news-agent/internal/summarizer"
	"news-agent/internal/telegram"

	"github.com/robfig/cron/v3"
)

func main() {
	var (
		configFile = flag.String("config", "config.yaml", "Path to config file")
		runOnce    = flag.Bool("once", false, "Run once and exit (for testing)")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize storage
	store := storage.NewStore("sent_articles.json")
	defer store.Cleanup()

	// Initialize Telegram client
	telegramChatID := cfg.Telegram.ChatID
	if telegramChatID == 0 {
		chatIDEnv := os.Getenv("TELEGRAM_CHAT_ID")
		if chatIDEnv != "" {
			chatID, _ := strconv.ParseInt(chatIDEnv, 10, 64)
			telegramChatID = chatID
		}
	}

	if telegramChatID == 0 {
		log.Fatalf("TELEGRAM_CHAT_ID not set")
	}

	tgClient, err := telegram.NewClient(cfg.Telegram.BotToken, telegramChatID)
	if err != nil {
		log.Fatalf("Failed to create Telegram client: %v", err)
	}

	// Initialize news fetcher
	newsFetcher := news.NewFetcher(cfg.NewsAPI.APIKey, cfg.RSS.Feeds, cfg.Preferences.Topics)

	// Initialize summarizer
	summ := summarizer.NewSummarizer(cfg.OpenAI.APIKey, cfg.OpenAI.Model)

	// Job function
	job := func() {
		if err := runNewsJob(context.Background(), newsFetcher, summ, tgClient, store, cfg.Preferences.NumArticles, cfg.Preferences.Topics); err != nil {
			log.Printf("Error running news job: %v", err)
		}
	}

	if *runOnce {
		job()
		return
	}

	// Run once immediately for testing
	job()

	// Schedule daily at 8 AM India Standard Time.
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Fatalf("Failed to load Asia/Kolkata timezone: %v", err)
	}
	c := cron.New(cron.WithLocation(ist))
	_, err = c.AddFunc("0 8 * * *", job)
	if err != nil {
		log.Fatalf("Failed to schedule cron job: %v", err)
	}

	log.Println("News agent started. Running test job immediately and then scheduled daily at 8 AM IST.")
	log.Println("Press Ctrl+C to stop.")

	c.Start()
	select {} // Keep running
}

// runNewsJob executes the daily news delivery job
func runNewsJob(ctx context.Context, newsFetcher *news.Fetcher, summ *summarizer.Summarizer, tgClient *telegram.Client, store *storage.Store, numArticles int, topics []string) error {
	log.Println("Starting news job...")

	// Calculate number of articles dynamically: 4 per topic
	calculatedNumArticles := len(topics) * 4
	if calculatedNumArticles > 0 {
		numArticles = calculatedNumArticles
	}
	log.Printf("Fetching %d articles (%d topics × 4 articles per topic)", numArticles, len(topics))

	// Fetch articles
	articles, err := newsFetcher.FetchAll()
	if err != nil {
		return fmt.Errorf("failed to fetch articles: %w", err)
	}

	if len(articles) == 0 {
		return fmt.Errorf("no articles fetched")
	}

	// Filter by preferences
	filteredArticles := newsFetcher.FilterByPreference(articles)
	if len(filteredArticles) == 0 {
		log.Println("No articles matched topic filters; using all articles")
		filteredArticles = articles
	} else {
		log.Printf("Using %d filtered articles out of %d total", len(filteredArticles), len(articles))
	}

	// Deduplicate and filter already sent
	var uniqueArticles []news.Article
	for _, article := range filteredArticles {
		if !store.IsSent(article.Hash) {
			uniqueArticles = append(uniqueArticles, article)
		}
	}

	if len(uniqueArticles) == 0 {
		log.Println("No new articles to send")
		return nil
	}
	uniqueArticles = selectArticlesPerTopic(uniqueArticles, topics, 4)
	if len(uniqueArticles) == 0 {
		log.Println("No new articles were available for the configured topics")
		return nil
	}

	// Extract titles and descriptions for summarization
	titles := make([]string, len(uniqueArticles))
	descriptions := make([]string, len(uniqueArticles))
	for i, article := range uniqueArticles {
		titles[i] = article.Title
		descriptions[i] = article.Description
	}

	// Summarize and curate
	summaries, dailySummary, err := summ.SummarizeArticles(ctx, titles, descriptions, topics, numArticles)
	if err != nil {
		return fmt.Errorf("failed to summarize articles: %w", err)
	}

	// Attach URLs to summaries
	articlesWithURLs := make([]telegram.ArticleSummary, len(summaries))
	for i, summary := range summaries {
		articlesWithURLs[i].Summary = summary
		if i < len(uniqueArticles) {
			articlesWithURLs[i].URL = uniqueArticles[i].URL
		}
	}

	// Format and send
	message := telegram.FormatNewsWithLinks(articlesWithURLs, dailySummary)
	err = tgClient.SendMessage(message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Record only articles that were actually included in the Telegram digest.
	// The summarizer receives articles in this same order.
	for i := 0; i < len(summaries) && i < len(uniqueArticles); i++ {
		article := uniqueArticles[i]
		store.Add(article.Hash, article.URL, article.Title, 0)
	}

	log.Printf("Successfully sent %d articles to Telegram", len(summaries))
	return nil
}

// selectArticlesPerTopic selects up to perTopic new articles for each configured
// topic. An article can appear in only one topic's allocation.
func selectArticlesPerTopic(articles []news.Article, topics []string, perTopic int) []news.Article {
	if len(topics) == 0 || perTopic <= 0 {
		return articles
	}

	selected := make([]news.Article, 0, len(topics)*perTopic)
	used := make(map[string]bool)
	for _, topic := range topics {
		count := 0
		for _, article := range articles {
			if count == perTopic || used[article.Hash] {
				continue
			}
			content := strings.ToLower(article.Title + " " + article.Description)
			if strings.Contains(content, strings.ToLower(topic)) {
				selected = append(selected, article)
				used[article.Hash] = true
				count++
			}
		}
		if count < perTopic {
			log.Printf("Topic %q has only %d new matching articles (wanted %d)", topic, count, perTopic)
		}
	}
	return selected
}
