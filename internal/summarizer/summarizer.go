package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type Summarizer struct {
	apiKey string
	model  string
}

func NewSummarizer(apiKey string, model string) *Summarizer {
	return &Summarizer{
		apiKey: apiKey,
		model:  model,
	}
}

type openAIRequest struct {
	Model               string      `json:"model"`
	Messages            []openAIMsg `json:"messages"`
	MaxCompletionTokens int         `json:"max_completion_tokens,omitempty"`
}

type openAIMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (s *Summarizer) SummarizeArticles(ctx context.Context, titles []string, descriptions []string, preferences []string, requestedCount int) ([]string, string, error) {
	if len(titles) == 0 {
		return nil, "", fmt.Errorf("no articles to summarize")
	}

	numArticles := requestedCount
	if numArticles <= 0 {
		numArticles = len(titles)
	}
	if numArticles > len(titles) {
		numArticles = len(titles)
	}
	if numArticles > 20 {
		numArticles = 20 // Cap at 20 for API limits
	}

	return s.summarizeWithCount(ctx, titles[:numArticles], descriptions[:numArticles], preferences, numArticles)
}

func (s *Summarizer) summarizeWithCount(ctx context.Context, titles []string, descriptions []string, preferences []string, numArticles int) ([]string, string, error) {
	if len(titles) == 0 {
		return nil, "", fmt.Errorf("no articles to summarize")
	}

	if s.apiKey == "" {
		log.Println("OpenAI API key missing; using fallback summarization")
		return fallbackSummarize(titles, descriptions, preferences, numArticles)
	}

	articlesContent := "Here are the news articles:\n\n"
	for i, title := range titles {
		desc := ""
		if i < len(descriptions) {
			desc = descriptions[i]
		}
		articlesContent += fmt.Sprintf("%d. Title: %s\nDescription: %s\n\n", i+1, title, desc)
	}

	prefsContent := ""
	if len(preferences) > 0 {
		prefsContent = fmt.Sprintf("User interests: %v\n\n", preferences)
	}

	prompt := fmt.Sprintf(`%sYou are a news curator. Analyze all these articles and:
1. Select exactly %d articles with balanced coverage from the provided topics
   - Pick exactly %d articles from each topic when that many relevant articles are available
   - Ensure representation of all user interests
2. For each selected article, write a 1-2 sentence summary
3. Provide an overall summary of today's key news

%s

Format your response EXACTLY as:
`, prefsContent, numArticles, numArticles/len(preferences), articlesContent)

	// Generate ARTICLE N: [Summary] lines dynamically
	for i := 1; i <= numArticles; i++ {
		prompt += fmt.Sprintf("ARTICLE %d: [Summary]\n", i)
	}

	prompt += "\nDAILY SUMMARY: [Overall summary]"

	reqBody := openAIRequest{
		Model: s.model,
		Messages: []openAIMsg{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxCompletionTokens: 1000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("OpenAI request marshal failed: %v; using fallback summarization", err)
		return fallbackSummarize(titles, descriptions, preferences, numArticles)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("OpenAI request creation failed: %v; using fallback summarization", err)
		return fallbackSummarize(titles, descriptions, preferences, numArticles)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("OpenAI API call failed: %v; using fallback summarization", err)
		return fallbackSummarize(titles, descriptions, preferences, numArticles)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("OpenAI response read failed: %v; using fallback summarization", err)
		return fallbackSummarize(titles, descriptions, preferences, numArticles)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("OpenAI API returned status %s: %s; using fallback summarization", resp.Status, strings.TrimSpace(string(body)))
		return fallbackSummarize(titles, descriptions, preferences, numArticles)
	}

	var aiResp openAIResponse
	err = json.Unmarshal(body, &aiResp)
	if err != nil || len(aiResp.Choices) == 0 {
		log.Printf("OpenAI response parse failed: %v; using fallback summarization", err)
		return fallbackSummarize(titles, descriptions, preferences, numArticles)
	}

	response := aiResp.Choices[0].Message.Content
	summaries, dailySummary := parseResponse(response)

	if len(summaries) < numArticles {
		log.Printf("OpenAI returned %d articles but expected %d. Response: %s", len(summaries), numArticles, response[:min(200, len(response))])
	}
	log.Printf("OpenAI selected %d articles for the digest (requested %d)", len(summaries), numArticles)

	return summaries, dailySummary, nil
}

func fallbackSummarize(titles []string, descriptions []string, preferences []string, requestedCount int) ([]string, string, error) {
	var summaries []string
	maxArticles := requestedCount
	if maxArticles <= 0 {
		maxArticles = len(titles)
	}
	if len(titles) < maxArticles {
		maxArticles = len(titles)
	}

	for i := 0; i < maxArticles; i++ {
		desc := titles[i]
		if i < len(descriptions) && len(descriptions[i]) > 0 {
			desc = descriptions[i][:min(200, len(descriptions[i]))]
		}
		summaries = append(summaries, desc)
	}

	dailySummary := fmt.Sprintf("Top news covering: %s", strings.Join(preferences, ", "))

	return summaries, dailySummary, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseResponse(response string) ([]string, string) {
	var summaries []string
	var dailySummary string

	if idx := strings.Index(response, "DAILY SUMMARY:"); idx >= 0 {
		dailySummary = strings.TrimSpace(response[idx+14:])
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ARTICLE") && len(line) > 10 {
			content := strings.TrimPrefix(line, "ARTICLE")
			for i := 1; i <= 10; i++ {
				content = strings.TrimPrefix(content, fmt.Sprintf("%d:", i))
			}
			content = strings.TrimSpace(content)
			if content != "" {
				summaries = append(summaries, content)
			}
		}
	}

	if dailySummary == "" {
		dailySummary = "News digest compiled."
	}

	return summaries, dailySummary
}

type ArticleWithURL struct {
	Summary string
	URL     string
}

func ParseResponseWithIndexes(response string) ([]ArticleWithURL, string) {
	var articlesWithURLs []ArticleWithURL
	var dailySummary string

	if idx := strings.Index(response, "DAILY SUMMARY:"); idx >= 0 {
		dailySummary = strings.TrimSpace(response[idx+14:])
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ARTICLE") && len(line) > 10 {
			content := strings.TrimPrefix(line, "ARTICLE")
			for i := 1; i <= 10; i++ {
				content = strings.TrimPrefix(content, fmt.Sprintf("%d:", i))
			}
			content = strings.TrimSpace(content)
			if content != "" {
				articlesWithURLs = append(articlesWithURLs, ArticleWithURL{
					Summary: content,
					URL:     "",
				})
			}
		}
	}

	if dailySummary == "" {
		dailySummary = "News digest compiled."
	}

	return articlesWithURLs, dailySummary
}
