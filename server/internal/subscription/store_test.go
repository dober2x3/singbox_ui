package subscription

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFileStore_SaveAndLoad verifies that saving a subscription and loading it back returns the same data.
func TestFileStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	data, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected error loading empty store: %v", err)
	}
	if len(data.Subscriptions) != 0 {
		t.Errorf("expected 0 subscriptions, got %d", len(data.Subscriptions))
	}

	entry := SubscriptionEntry{
		ID:   "test-1",
		Name: "Test Sub",
		URL:  "https://example.com/sub",
	}
	data.Subscriptions = append(data.Subscriptions, entry)

	if err := store.Save(*data); err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected error loading: %v", err)
	}
	if len(loaded.Subscriptions) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(loaded.Subscriptions))
	}
	if loaded.Subscriptions[0].ID != "test-1" {
		t.Errorf("expected ID test-1, got %s", loaded.Subscriptions[0].ID)
	}
}

// TestFileStore_FileNotExist verifies that Load returns an empty list when the JSON file does not exist.
func TestFileStore_FileNotExist(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	store := NewFileStore(dir)

	data, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Subscriptions == nil {
		t.Error("expected non-nil subscriptions slice")
	}
}

// TestFileStore_FilePath verifies that filePath returns the correct subscription.json path in the base directory.
func TestFileStore_FilePath(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	if store.filePath() != filepath.Join(dir, "subscription.json") {
		t.Errorf("unexpected file path: %s", store.filePath())
	}
}

// TestFileStore_DefaultDir verifies that NewFileStore with empty string defaults to the current working directory.
func TestFileStore_DefaultDir(t *testing.T) {
	originalWd, _ := os.Getwd()
	store := NewFileStore("")
	expected := filepath.Join(originalWd, "subscription.json")
	if store.filePath() != expected {
		t.Errorf("expected path %s, got %s", expected, store.filePath())
	}
}
