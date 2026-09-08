package storage

import (
	"path/filepath"
	"testing"
)

func TestAddSavesWithoutDeadlock(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "sent_articles.json"))

	if err := store.Add("hash-1", "https://example.com/story", "Example story", 123); err != nil {
		t.Fatalf("Add returned error: %v", err)
	}

	reloaded := NewStore(store.filePath)
	if !reloaded.IsSent("hash-1") {
		t.Fatal("expected saved article to be present after reload")
	}
}

func TestCleanupSavesWithoutDeadlock(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "sent_articles.json"))

	if err := store.Cleanup(); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
}
