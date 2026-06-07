package resourcecheck

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStore_CreateAndQuery(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	result := CheckResult{
		Resource:  "youtube",
		Tag:       "node-1",
		Status:    "ok",
		LatencyMs: 150,
		HTTPCode:  200,
		CheckedAt: now,
	}

	if err := store.SaveResult(result); err != nil {
		t.Fatalf("SaveResult() error = %v", err)
	}

	// Query latest results
	results, err := store.GetLatestResults()
	if err != nil {
		t.Fatalf("GetLatestResults() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Resource != "youtube" {
		t.Errorf("expected resource youtube, got %s", results[0].Resource)
	}
	if results[0].Status != "ok" {
		t.Errorf("expected status ok, got %s", results[0].Status)
	}

	// Save a newer result for same resource+tag
	result2 := CheckResult{
		Resource:  "youtube",
		Tag:       "node-1",
		Status:    "timeout",
		LatencyMs: -1,
		CheckedAt: time.Now().UTC().Add(time.Second).Format(time.RFC3339),
	}
	if err := store.SaveResult(result2); err != nil {
		t.Fatalf("SaveResult() error = %v", err)
	}

	// Latest should return only the newest
	results, err = store.GetLatestResults()
	if err != nil {
		t.Fatalf("GetLatestResults() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 latest result, got %d", len(results))
	}
	if results[0].Status != "timeout" {
		t.Errorf("expected latest status timeout, got %s", results[0].Status)
	}

	// History should return both
	history, err := store.GetHistory("youtube", "node-1", 10)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
}

func TestStore_GetResultsForTag(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	store.SaveResult(CheckResult{Resource: "youtube", Tag: "node-1", Status: "ok", LatencyMs: 100, CheckedAt: now})
	store.SaveResult(CheckResult{Resource: "telegram", Tag: "node-1", Status: "ok", LatencyMs: 50, CheckedAt: now})
	store.SaveResult(CheckResult{Resource: "youtube", Tag: "node-2", Status: "timeout", LatencyMs: -1, CheckedAt: now})

	results, err := store.GetResultsForTag("node-1")
	if err != nil {
		t.Fatalf("GetResultsForTag() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for node-1, got %d", len(results))
	}

	tags, err := store.GetTags()
	if err != nil {
		t.Fatalf("GetTags() error = %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
}

func TestStore_EmptyDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	results, err := store.GetLatestResults()
	if err != nil {
		t.Fatalf("GetLatestResults() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results in empty db, got %d", len(results))
	}
}
