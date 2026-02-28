package config

import (
	"encoding/json"
	"io"
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

// NewConfigManager initializes a manager, loading the config file from disk or creating default if not found
func NewConfigManager() (*ConfigManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(home, "Library", "Application Support", "chrome-url-tracker")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.json")

	manager := &ConfigManager{
		path:   configPath,
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

	file, err := os.Open(m.path)
	if err != nil {
		return err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, &m.config)
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
