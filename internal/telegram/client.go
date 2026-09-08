package telegram

import (
	"fmt"
	"regexp"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

func NewClient(botToken string, chatID int64) (*Client, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	return &Client{
		bot:    bot,
		chatID: chatID,
	}, nil
}

func (c *Client) SendMessage(message string) error {
	msg := tgbotapi.NewMessage(c.chatID, message)
	msg.ParseMode = "HTML"

	_, err := c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func FormatNews(articles []string, dailySummary string) string {
	message := "<b>📰 Daily News Digest</b>\n"
	message += fmt.Sprintf("<i>%s</i>\n\n", dailySummary)

	for _, article := range articles {
		message += fmt.Sprintf("%s\n\n", article)
	}

	message += "<i>Stay informed! 🌍</i>"
	return message
}

type ArticleSummary struct {
	Summary string
	URL     string
}

func FormatNewsWithLinks(articles []ArticleSummary, dailySummary string) string {
	message := "<b>📰 Daily News Digest</b>\n"
	message += fmt.Sprintf("<i>%s</i>\n\n", dailySummary)

	for i, article := range articles {
		// Strip HTML tags like <img> from summary
		summary := stripHTMLTags(article.Summary)
		message += fmt.Sprintf("%d. %s", i+1, summary)
		if article.URL != "" {
			message += fmt.Sprintf("\n🔗 <a href=\"%s\">Read more</a>", article.URL)
		}
		message += "\n\n"
	}

	message += "<i>Stay informed! 🌍</i>"
	return message
}

func stripHTMLTags(text string) string {
	// Remove img tags
	re := regexp.MustCompile(`<img[^>]*>`)
	text = re.ReplaceAllString(text, "")
	// Remove script tags
	re = regexp.MustCompile(`<script[^>]*>.*?</script>`)
	text = re.ReplaceAllString(text, "")
	// Remove style tags
	re = regexp.MustCompile(`<style[^>]*>.*?</style>`)
	text = re.ReplaceAllString(text, "")
	return text
}
