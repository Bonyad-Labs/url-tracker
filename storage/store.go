// Package storage handles persistent storage of URL entries and whitelisted domains.
// It provides a thread-safe Store that persists data as JSON with atomic write guarantees.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	path            string
	entries         []Entry
	excludedDomains []WhitelistEntry
	mu              sync.RWMutex
}

type storageData struct {
	Entries         []Entry          `json:"entries"`
	ExcludedDomains []WhitelistEntry `json:"excluded_domains"`
}

// NewStore initializes a new Store at the given path.
// It supports tilde expansion (e.g., "~/Documents/urls.json") and automatically
// handles file initialization, default exclusions, and corruption recovery via backups.
func NewStore(path string) (*Store, error) {
	// Expand tilde if present
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	s := &Store{path: path}
	err := s.load()
	if err != nil && !os.IsNotExist(err) {
		// Attempt to handle corruption by backing up
		backupPath := fmt.Sprintf("%s.backup.%d", path, time.Now().Unix())
		_ = os.Rename(path, backupPath)
		// Initialize empty if load failed (after backup attempt)
		s.entries = []Entry{}
		s.excludedDomains = getDefaultExclusions()
		_ = s.Save()
		return s, fmt.Errorf("storage corrupted, backup created at %s, reset to empty", backupPath)
	}

	if os.IsNotExist(err) {
		// Initialize empty file
		s.entries = []Entry{}
		s.excludedDomains = getDefaultExclusions()
		err = s.Save()
		if err != nil {
			return nil, err
		}
	}

	// Ensure defaults if empty
	if len(s.excludedDomains) == 0 {
		s.excludedDomains = getDefaultExclusions()
		_ = s.Save()
	}

	return s, nil
}

func (s *Store) load() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadUnlocked()
}

func (s *Store) loadUnlocked() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	// We use a temporary struct to detect the format of excluded_domains
	var sd struct {
		Entries         []Entry           `json:"entries"`
		ExcludedDomains []json.RawMessage `json:"excluded_domains"`
	}

	if err := json.Unmarshal(data, &sd); err != nil {
		return err
	}

	s.entries = sd.Entries
	s.excludedDomains = []WhitelistEntry{}

	for _, raw := range sd.ExcludedDomains {
		// Attempt to parse as new structured entry
		var entry WhitelistEntry
		if err := json.Unmarshal(raw, &entry); err == nil && entry.Value != "" {
			s.excludedDomains = append(s.excludedDomains, entry)
			continue
		}

		// Fallback: Attempt to parse as legacy string
		var legacyValue string
		if err := json.Unmarshal(raw, &legacyValue); err == nil {
			entryType := "domain"
			if strings.Contains(legacyValue, "://") {
				entryType = "url"
			}
			s.excludedDomains = append(s.excludedDomains, WhitelistEntry{
				Value:     legacyValue,
				Type:      entryType,
				Timestamp: time.Now().Unix(),
			})
		}
	}

	return nil
}

// Save persists the current state of the Store to disk.
// It performs an atomic write by writing to a temporary file and renaming it,
// ensuring that data is not lost or corrupted during an interrupted write.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked()
}

func (s *Store) saveUnlocked() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	sd := storageData{
		Entries:         s.entries,
		ExcludedDomains: s.excludedDomains,
	}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}

// AddEntry adds a new URL entry to the store and persists it to disk.
// It automatically sets the current Unix timestamp on the entry.
func (s *Store) AddEntry(e Entry) error {
	e.Timestamp = time.Now().Unix()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
	return s.saveUnlocked()
}

// GetEntries returns a thread-safe copy of all saved URL entries.
func (s *Store) GetEntries() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]Entry, len(s.entries))
	copy(res, s.entries)
	return res
}

// EntryExists checks if a specific URL has already been saved in the store.
func (s *Store) EntryExists(url string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.entries {
		if e.URL == url {
			return true
		}
	}
	return false
}

// GetExcludedDomains returns a copy of all whitelisted entries.
func (s *Store) GetExcludedDomains() []WhitelistEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]WhitelistEntry, len(s.excludedDomains))
	copy(res, s.excludedDomains)
	return res
}

// SearchEntries performs a case-insensitive substring search across all fields of all entries.
// Fields searched include URL, Title, Description, Tags, and Category.
func (s *Store) SearchEntries(query string) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if query == "" {
		return s.entries
	}

	query = strings.ToLower(query)
	var results []Entry
	for _, e := range s.entries {
		if matchesQuery(e, query) {
			results = append(results, e)
		}
	}
	return results
}

// AddExcludedDomain adds a domain or URL to the exclusion list.
// Prevents duplicate entries and persists the updated list to disk.
func (s *Store) AddExcludedDomain(domain string) error {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.excludedDomains {
		if d.Value == domain {
			return nil
		}
	}

	entryType := "domain"
	if strings.Contains(domain, "://") {
		entryType = "url"
	}

	s.excludedDomains = append(s.excludedDomains, WhitelistEntry{
		Value:     domain,
		Type:      entryType,
		Timestamp: time.Now().Unix(),
	})
	return s.saveUnlocked()
}

// RemoveExcludedDomain removes a domain or URL from the exclusion list.
// If the domain exists, the list is updated and persisted to disk.
func (s *Store) RemoveExcludedDomain(domain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newDomains := []WhitelistEntry{}
	found := false
	for _, d := range s.excludedDomains {
		if d.Value == domain {
			found = true
			continue
		}
		newDomains = append(newDomains, d)
	}

	if !found {
		return nil
	}

	s.excludedDomains = newDomains
	return s.saveUnlocked()
}

// IsExcluded returns true if the given URL matches any whitelisted domain or URL fragment.
// The check is case-insensitive.
func (s *Store) IsExcluded(url string) bool {
	url = strings.ToLower(url)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, entry := range s.excludedDomains {
		if strings.Contains(url, entry.Value) {
			return true
		}
	}
	return false
}

func matchesQuery(e Entry, query string) bool {
	if strings.Contains(strings.ToLower(e.URL), query) ||
		strings.Contains(strings.ToLower(e.Title), query) ||
		strings.Contains(strings.ToLower(e.Description), query) ||
		strings.Contains(strings.ToLower(e.Category), query) {
		return true
	}
	for _, tag := range e.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}
