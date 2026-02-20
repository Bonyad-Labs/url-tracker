// Package storage handles persistent storage of URL entries and whitelisted domains.
// It provides a thread-safe Store that persists data as JSON with atomic write guarantees.
package storage

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

func getDefaultExclusions() []WhitelistEntry {
	defaults := []string{"localhost", "127.0.0.1", "accounts.google.com", "login.microsoftonline.com"}
	res := make([]WhitelistEntry, len(defaults))
	for i, d := range defaults {
		res[i] = WhitelistEntry{
			Value:     d,
			Type:      "domain",
			Timestamp: time.Now().Unix(),
		}
	}
	return res
}

// DefaultPath returns the standard macOS location for application data:
// ~/Library/Application Support/chrome-url-tracker/chrome-urls.db
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "chrome-urls.db" // Fallback to current dir
	}
	return filepath.Join(home, "Library", "Application Support", "chrome-url-tracker", "chrome-urls.db")
}

// Entry represents a single saved URL with its associated metadata.
type Entry struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Category    string   `json:"category"`
	Timestamp   int64    `json:"timestamp"` // Unix timestamp of when the entry was saved
}

// WhitelistEntry represents a domain or URL exclusion with metadata.
type WhitelistEntry struct {
	Value     string `json:"value"`
	Type      string `json:"type"`      // "domain" or "url"
	Timestamp int64  `json:"timestamp"` // Unix timestamp of when it was added
}

// Store manages the collection of saved URLs and whitelisted domains.
// It is thread-safe and safe for concurrent use by multiple goroutines.
type Store struct {
	path string
	db   *sql.DB
	mu   sync.RWMutex
}

// NewStore initializes a new Store. If path is empty, it uses the standard macOS
// Application Support location and handles automatic relocation from ~/Documents if data exists there.
func NewStore(path string) (*Store, error) {
	isDefault := path == ""
	if isDefault {
		path = DefaultPath()
	}

	// Expand tilde if present
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	// Change extension if it's .json to .db if user provided default
	if strings.HasSuffix(path, ".json") {
		path = strings.TrimSuffix(path, ".json") + ".db"
	}

	// Automatic Relocation from Documents if using default path
	if isDefault {
		home, _ := os.UserHomeDir()
		oldDB := filepath.Join(home, "Documents", "chrome-urls.db")
		oldMigrated := filepath.Join(home, "Documents", "chrome-urls.json.migrated")

		// If new DB doesn't exist but old one does, move it!
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if _, err := os.Stat(oldDB); err == nil {
				_ = os.MkdirAll(filepath.Dir(path), 0755)
				if err := os.Rename(oldDB, path); err == nil {
					log.Printf("Relocated database from %s to %s", oldDB, path)
					// Also try moving the migrated JSON if it exists
					_ = os.Rename(oldMigrated, strings.TrimSuffix(path, ".db")+".json.migrated")
				}
			}
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Performance and Concurrency Tuning
	// Enable WAL mode for concurrent read/write and set a busy timeout for locks
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA busy_timeout=5000;") // 5 seconds

	s := &Store{path: path, db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS entries (
			url TEXT PRIMARY KEY,
			title TEXT,
			description TEXT,
			category TEXT,
			timestamp INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			url TEXT,
			tag TEXT,
			FOREIGN KEY(url) REFERENCES entries(url) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS whitelist (
			value TEXT PRIMARY KEY,
			type TEXT,
			timestamp INTEGER
		)`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}

	// Insert default whitelist if empty
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM whitelist").Scan(&count)
	if count == 0 {
		defaults := getDefaultExclusions()
		for _, d := range defaults {
			_, _ = s.db.Exec("INSERT OR IGNORE INTO whitelist (value, type, timestamp) VALUES (?, ?, ?)",
				d.Value, d.Type, d.Timestamp)
		}
	}

	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Save is a no-op for SQLite as it auto-persists, kept for API compatibility.
func (s *Store) Save() error {
	return nil
}

// AddEntry adds a new URL entry to the store and persists it to disk.
func (s *Store) AddEntry(e Entry) error {
	e.Timestamp = time.Now().Unix()
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT OR REPLACE INTO entries (url, title, description, category, timestamp) VALUES (?, ?, ?, ?, ?)",
		e.URL, e.Title, e.Description, e.Category, e.Timestamp)
	if err != nil {
		return err
	}

	// Re-insert tags
	_, _ = tx.Exec("DELETE FROM tags WHERE url = ?", e.URL)
	for _, t := range e.Tags {
		_, _ = tx.Exec("INSERT INTO tags (url, tag) VALUES (?, ?)", e.URL, t)
	}

	return tx.Commit()
}

// GetEntries returns all saved URL entries.
func (s *Store) GetEntries() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT url, title, description, category, timestamp FROM entries ORDER BY timestamp DESC")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var res []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.URL, &e.Title, &e.Description, &e.Category, &e.Timestamp); err == nil {
			res = append(res, e)
		}
	}

	// Fetch all tags for these entries in a single batch to avoid N+1 queries
	if len(res) > 0 {
		tagRows, err := s.db.Query("SELECT url, tag FROM tags")
		if err == nil {
			defer tagRows.Close()
			tagMap := make(map[string][]string)
			for tagRows.Next() {
				var u, t string
				if err := tagRows.Scan(&u, &t); err == nil {
					tagMap[u] = append(tagMap[u], t)
				}
			}
			for i := range res {
				res[i].Tags = tagMap[res[i].URL]
			}
		}
	}

	return res
}

// EntryExists checks if a specific URL has already been saved in the store.
func (s *Store) EntryExists(url string) bool {
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM entries WHERE url = ?", url).Scan(&count)
	return count > 0
}

// GetExcludedDomains returns a copy of all whitelisted entries.
func (s *Store) GetExcludedDomains() []WhitelistEntry {
	rows, err := s.db.Query("SELECT value, type, timestamp FROM whitelist ORDER BY timestamp DESC")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var res []WhitelistEntry
	for rows.Next() {
		var w WhitelistEntry
		if err := rows.Scan(&w.Value, &w.Type, &w.Timestamp); err == nil {
			res = append(res, w)
		}
	}
	return res
}

// SearchEntries performs a case-insensitive substring search across all fields of all entries.
func (s *Store) SearchEntries(query string) []Entry {
	if query == "" {
		return s.GetEntries()
	}

	q := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.Query(`
		SELECT url, title, description, category, timestamp FROM entries 
		WHERE LOWER(url) LIKE ? 
		   OR LOWER(title) LIKE ? 
		   OR LOWER(description) LIKE ? 
		   OR LOWER(category) LIKE ?
		   OR url IN (SELECT url FROM tags WHERE LOWER(tag) LIKE ?)
		ORDER BY timestamp DESC`, q, q, q, q, q)

	if err != nil {
		return nil
	}
	defer rows.Close()

	var res []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.URL, &e.Title, &e.Description, &e.Category, &e.Timestamp); err == nil {
			res = append(res, e)
		}
	}

	// Fetch tags batch
	if len(res) > 0 {
		tagRows, err := s.db.Query("SELECT url, tag FROM tags")
		if err == nil {
			defer tagRows.Close()
			tagMap := make(map[string][]string)
			for tagRows.Next() {
				var u, t string
				if err := tagRows.Scan(&u, &t); err == nil {
					tagMap[u] = append(tagMap[u], t)
				}
			}
			for i := range res {
				res[i].Tags = tagMap[res[i].URL]
			}
		}
	}

	return res
}

// AddExcludedDomain adds a domain or URL to the exclusion list.
func (s *Store) AddExcludedDomain(domain string) error {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return nil
	}

	entryType := "domain"
	if strings.Contains(domain, "://") {
		entryType = "url"
	}

	_, err := s.db.Exec("INSERT OR IGNORE INTO whitelist (value, type, timestamp) VALUES (?, ?, ?)",
		domain, entryType, time.Now().Unix())
	return err
}

// RemoveExcludedDomain removes a domain or URL from the exclusion list.
func (s *Store) RemoveExcludedDomain(domain string) error {
	_, err := s.db.Exec("DELETE FROM whitelist WHERE value = ?", domain)
	return err
}

// IsExcluded returns true if the given URL matches any whitelisted domain or URL fragment.
func (s *Store) IsExcluded(url string) bool {
	url = strings.ToLower(url)

	// Optimized SQL check using pattern matching:
	// INSTR(string, substring) returns > 0 if found.
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM whitelist WHERE INSTR(?, value) > 0)", url).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}
