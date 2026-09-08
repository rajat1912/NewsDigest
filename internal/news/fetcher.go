package news

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Article struct {
	Title       string
	Description string
	URL         string
	Source      string
	PublishedAt time.Time
	Hash        string
}

type Fetcher struct {
	newsAPIKey string
	rssFeeds   []string
	topics     []string
}

func NewFetcher(newsAPIKey string, rssFeeds []string, topics []string) *Fetcher {
	return &Fetcher{
		newsAPIKey: newsAPIKey,
		rssFeeds:   rssFeeds,
		topics:     topics,
	}
}

func (f *Fetcher) FetchAll() ([]Article, error) {
	var articles []Article

	// Fetch from RSS feeds
	rssCount := 0
	for _, feed := range f.rssFeeds {
		rssArticles, err := f.fetchFromRSS(feed)
		if err != nil {
			fmt.Printf("Error fetching RSS feed %s: %v\n", feed, err)
		} else {
			articles = append(articles, rssArticles...)
			rssCount += len(rssArticles)
		}
	}

	// Fetch from NewsAPI if key is provided
	newsAPICount := 0
	if f.newsAPIKey != "" {
		newsAPIArticles, err := f.fetchFromNewsAPI()
		if err != nil {
			log.Printf("Error fetching from NewsAPI: %v", err)
		} else {
			articles = append(articles, newsAPIArticles...)
			newsAPICount = len(newsAPIArticles)
		}
	}

	log.Printf("Fetched %d articles from RSS feeds, %d from NewsAPI (total: %d)", rssCount, newsAPICount, len(articles))
	return articles, nil
}

type rss struct {
	XMLName xml.Name `xml:"rss"`
	Channel channel  `xml:"channel"`
}

type channel struct {
	Title string `xml:"title"`
	Items []item `xml:"item"`
}

type item struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
}

func (f *Fetcher) fetchFromRSS(feedURL string) ([]Article, error) {
	resp, err := http.Get(feedURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rssFeed rss
	err = xml.NewDecoder(resp.Body).Decode(&rssFeed)
	if err != nil {
		return nil, err
	}

	var articles []Article
	for _, i := range rssFeed.Channel.Items {
		article := Article{
			Title:       i.Title,
			Description: i.Description,
			URL:         i.Link,
			Source:      rssFeed.Channel.Title,
		}

		if i.PubDate != "" {
			pubDate, err := time.Parse(time.RFC1123Z, i.PubDate)
			if err != nil {
				pubDate = time.Now()
			}
			article.PublishedAt = pubDate
		} else {
			article.PublishedAt = time.Now()
		}

		article.Hash = generateHash(article.URL)
		articles = append(articles, article)
	}

	return articles, nil
}

type newsAPIResponse struct {
	Articles []newsAPIArticle `json:"articles"`
}

type newsAPIArticle struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Source      struct {
		Name string `json:"name"`
	} `json:"source"`
	PublishedAt string `json:"publishedAt"`
}

func (f *Fetcher) fetchFromNewsAPI() ([]Article, error) {
	if f.newsAPIKey == "" {
		return nil, fmt.Errorf("newsapi key not configured")
	}

	// Build query from topics
	query := strings.Join(f.topics, " OR ")
	if query == "" {
		query = "news"
	}

	// URL encode the query
	encodedQuery := url.QueryEscape(query)
	apiURL := fmt.Sprintf("https://newsapi.org/v2/everything?q=%s&sortBy=publishedAt&language=en&apiKey=%s", encodedQuery, f.newsAPIKey)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("newsapi returned status %d", resp.StatusCode)
	}

	var apiResp newsAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		return nil, err
	}

	var articles []Article
	for _, a := range apiResp.Articles {
		article := Article{
			Title:       a.Title,
			Description: a.Description,
			URL:         a.URL,
			Source:      "NewsAPI - " + a.Source.Name,
		}

		if a.PublishedAt != "" {
			pubDate, err := time.Parse(time.RFC3339, a.PublishedAt)
			if err != nil {
				pubDate = time.Now()
			}
			article.PublishedAt = pubDate
		} else {
			article.PublishedAt = time.Now()
		}

		article.Hash = generateHash(article.URL)
		articles = append(articles, article)
	}

	return articles, nil
}

func generateHash(url string) string {
	hash := strings.ReplaceAll(url, "https://", "")
	hash = strings.ReplaceAll(hash, "http://", "")
	if len(hash) > 50 {
		return hash[:50]
	}
	return hash
}

func (f *Fetcher) FilterByPreference(articles []Article) []Article {
	if len(f.topics) == 0 {
		return articles
	}

	var filtered []Article
	topicCounts := make(map[string]int)

	for _, article := range articles {
		content := strings.ToLower(article.Title + " " + article.Description)
		for _, topic := range f.topics {
			if strings.Contains(content, strings.ToLower(topic)) {
				filtered = append(filtered, article)
				topicCounts[topic]++
				break
			}
		}
	}

	for topic, count := range topicCounts {
		log.Printf("Topic '%s': %d articles matched", topic, count)
	}

	log.Printf("Filtered %d articles from %d total by topics: %v", len(filtered), len(articles), f.topics)
	return filtered
}
