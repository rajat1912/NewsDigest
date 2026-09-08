package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type SentArticle struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Hash      string    `json:"hash"`
	SentAt    time.Time `json:"sent_at"`
	MessageID int       `json:"message_id,omitempty"`
}

type Store struct {
	filePath string
	articles map[string]*SentArticle
	mu       sync.RWMutex
	maxAge   time.Duration
}

func NewStore(filePath string) *Store {
	store := &Store{
		filePath: filePath,
		articles: make(map[string]*SentArticle),
		maxAge:   24 * time.Hour * 30,
	}

	_ = store.Load()
	return store
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read storage file: %w", err)
	}

	var articles []SentArticle
	err = json.Unmarshal(data, &articles)
	if err != nil {
		return fmt.Errorf("failed to parse storage file: %w", err)
	}

	for i := range articles {
		s.articles[articles[i].Hash] = &articles[i]
	}

	return nil
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	articles := make([]SentArticle, 0, len(s.articles))
	for _, article := range s.articles {
		articles = append(articles, *article)
	}

	data, err := json.MarshalIndent(articles, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal articles: %w", err)
	}

	err = os.WriteFile(s.filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write storage file: %w", err)
	}

	return nil
}

func (s *Store) IsSent(hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.articles[hash]
	return exists
}

func (s *Store) Add(hash string, url string, title string, messageID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.articles[hash] = &SentArticle{
		URL:       url,
		Title:     title,
		Hash:      hash,
		SentAt:    time.Now(),
		MessageID: messageID,
	}

	return s.saveLocked()
}

func (s *Store) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	toDelete := []string{}

	for hash, article := range s.articles {
		if now.Sub(article.SentAt) > s.maxAge {
			toDelete = append(toDelete, hash)
		}
	}

	for _, hash := range toDelete {
		delete(s.articles, hash)
	}

	return s.saveLocked()
}

func (s *Store) GetRecentSentCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	today := time.Now().Truncate(24 * time.Hour)
	count := 0

	for _, article := range s.articles {
		if article.SentAt.After(today) {
			count++
		}
	}

	return count
}
