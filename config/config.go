package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Config represents the application settings
type Config struct {
	PollingInterval int    `json:"polling_interval"`
	StoragePath     string `json:"storage_path"`
}

// DefaultConfig returns the default application configuration
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		PollingInterval: 1000,
		StoragePath:     filepath.Join(home, "Library", "Application Support", "chrome-url-tracker", "chrome-urls.db"),
	}
}

// ConfigManager handles thread-safe access to the application configuration
type ConfigManager struct {
	mu     sync.RWMutex
	config Config
	path   string
}

// NewConfigManager initializes a manager with the default path
func NewConfigManager() (*ConfigManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configDir := filepath.Join(home, "Library", "Application Support", "chrome-url-tracker")
	return NewConfigManagerAtPath(filepath.Join(configDir, "config.json"))
}

// NewConfigManagerAtPath initializes a manager at a specific path, useful for testing
func NewConfigManagerAtPath(path string) (*ConfigManager, error) {
	configDir := filepath.Dir(path)
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, err
	}

	manager := &ConfigManager{
		path:   path,
		config: DefaultConfig(),
	}

	err = manager.load()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Always save, so default file is created if it didn't exist
	_ = manager.Save()

	return manager, nil
}

// load reads the config.json file
func (m *ConfigManager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &m.config)
	if err != nil {
		// Corner case: Corrupted config file.
		// Log and continue with defaults to avoid blocking app start.
		log.Printf("Warning: Corrupted config file at %s, using defaults: %v", m.path, err)
		m.config = DefaultConfig()
		return nil
	}

	return nil
}

// Save writes the current configuration to disk
func (m *ConfigManager) Save() error {
	m.mu.RLock()
	configCopy := m.config
	m.mu.RUnlock()

	bytes, err := json.MarshalIndent(configCopy, "", "  ")
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return os.WriteFile(m.path, bytes, 0644)
}

// Get returns a copy of the current configuration
func (m *ConfigManager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// SetInterval updates the polling interval safely and persists it
func (m *ConfigManager) SetInterval(interval int) error {
	m.mu.Lock()
	m.config.PollingInterval = interval
	m.mu.Unlock()
	return m.Save()
}
