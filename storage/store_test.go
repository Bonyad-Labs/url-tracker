package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	tempDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(tempDir, "urls.db")
	store, err := NewStore(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatal(err)
	}
	return store, func() {
		store.Close()
		os.RemoveAll(tempDir)
	}
}

func TestCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	entry := Entry{
		URL:         "https://example.com/test",
		Title:       "Test Page",
		Description: "A description",
		Tags:        []string{"test", "tag"},
		Category:    "Testing",
	}

	// Test Add
	if err := store.AddEntry(entry); err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	// Test Exists
	if !store.EntryExists(entry.URL) {
		t.Errorf("EntryExists returned false for existing URL")
	}

	// Test Get
	entries := store.GetEntries()
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].URL != entry.URL || entries[0].Title != entry.Title {
		t.Errorf("retrieved entry does not match")
	}
	if len(entries[0].Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(entries[0].Tags))
	}

	// Test Update (AddEntry uses INSERT OR REPLACE)
	entry.Title = "Updated Title"
	if err := store.AddEntry(entry); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	entries = store.GetEntries()
	if len(entries) != 1 {
		t.Errorf("expected 1 entry after update, got %d", len(entries))
	}
	if entries[0].Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %s", entries[0].Title)
	}
}

func TestSearch(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_ = store.AddEntry(Entry{URL: "https://apple.com", Title: "Apple Home", Category: "Tech"})
	_ = store.AddEntry(Entry{URL: "https://google.com", Title: "Google Search", Category: "Search", Tags: []string{"search"}})
	_ = store.AddEntry(Entry{URL: "https://github.com", Title: "GitHub", Description: "Code repo", Tags: []string{"git"}})

	tests := []struct {
		query string
		want  int
	}{
		{"apple", 1},
		{"SEARCH", 1}, // Case insensitive
		{"code", 1},   // Description
		{"git", 1},    // Tags
		{"tech", 1},   // Category
		{"o", 3},      // Multiple (google, apple, github)
		{"xyz", 0},    // No match
		{"", 3},       // Empty query returns all
	}

	for _, tt := range tests {
		got := store.SearchEntries(tt.query)
		if len(got) != tt.want {
			t.Errorf("SearchEntries(%q) got %d, want %d", tt.query, len(got), tt.want)
		}
	}
}

func TestWhitelist(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Initial defaults should be present
	initial := store.GetExcludedDomains()
	if len(initial) == 0 {
		t.Error("expected default exclusions, got none")
	}

	// Test Add
	if err := store.AddExcludedDomain("youtube.com"); err != nil {
		t.Fatal(err)
	}
	if !store.IsExcluded("https://www.youtube.com/watch?v=123") {
		t.Error("youtube.com should be excluded")
	}

	// Test Partial Match / Subdomain
	if !store.IsExcluded("https://sub.youtube.com/test") {
		t.Error("subdomain of whitelisted domain should be excluded")
	}

	// Test URL Whitelisting
	if err := store.AddExcludedDomain("https://exclusive.com/page"); err != nil {
		t.Fatal(err)
	}
	if !store.IsExcluded("https://exclusive.com/page") {
		t.Error("specific URL should be excluded")
	}
	if store.IsExcluded("https://exclusive.com/other") {
		t.Error("other page on same domain should NOT be excluded if only specific URL whitelisted")
	}

	// Test Remove
	if err := store.RemoveExcludedDomain("youtube.com"); err != nil {
		t.Fatal(err)
	}
	if store.IsExcluded("https://youtube.com") {
		t.Error("youtube.com should NOT be excluded after removal")
	}
}

func TestRelocation(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "home_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempHome)

	// Mock HOME for this test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", origHome)

	// Create legacy database in Documents
	oldDir := filepath.Join(tempHome, "Documents")
	os.MkdirAll(oldDir, 0755)
	oldDB := filepath.Join(oldDir, "chrome-urls.db")
	db, err := sql.Open("sqlite", oldDB)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Initialize store with empty path (trigger default relocation)
	// We need to make sure we don't open the "real" Application Support
	s, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Verify file was moved to Application Support
	newPath := DefaultPath()
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Errorf("Database was not moved to %s", newPath)
	}
	if _, err := os.Stat(oldDB); err == nil {
		t.Errorf("Old database still exists at %s", oldDB)
	}
}

func TestConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "concurrency_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	dbPath := filepath.Join(tempDir, "concurrency.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	const numGoRoutines = 20
	const numOpsPerGoRoutine = 50
	var wg sync.WaitGroup
	wg.Add(numGoRoutines)

	for i := 0; i < numGoRoutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoRoutine; j++ {
				url := fmt.Sprintf("https://example.com/%d/%d", id, j)
				entry := Entry{
					URL:       url,
					Title:     "Test",
					Timestamp: time.Now().Unix(),
				}
				if err := store.AddEntry(entry); err != nil {
					t.Errorf("AddEntry failed: %v", err)
				}
				if ext := store.EntryExists(url); !ext {
					t.Errorf("Entry %s should exist", url)
				}
				// Concurrent search
				results := store.SearchEntries("example")
				if len(results) == 0 {
					t.Errorf("Search should return results")
				}
			}
		}(i)
	}

	wg.Wait()
}
