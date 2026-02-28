package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Test 1: NewConfigManagerAtPath creates default file if missing
	manager, err := NewConfigManagerAtPath(configPath)
	if err != nil {
		t.Fatalf("failed to create config manager: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	cfg := manager.Get()
	if cfg.PollingInterval != 1000 {
		t.Errorf("expected default polling interval 1000, got %d", cfg.PollingInterval)
	}

	// Test 2: SetInterval updates and saves
	err = manager.SetInterval(2000)
	if err != nil {
		t.Fatalf("failed to set interval: %v", err)
	}

	cfg = manager.Get()
	if cfg.PollingInterval != 2000 {
		t.Errorf("expected updated polling interval 2000, got %d", cfg.PollingInterval)
	}

	// Test 3: Load existing config
	manager2, err := NewConfigManagerAtPath(configPath)
	if err != nil {
		t.Fatalf("failed to reload config manager: %v", err)
	}

	if manager2.Get().PollingInterval != 2000 {
		t.Errorf("expected reloaded polling interval 2000, got %d", manager2.Get().PollingInterval)
	}
}

func TestConfigManagerCornerCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_corner_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test 1: Invalid JSON content
	configPath := filepath.Join(tempDir, "invalid.json")
	err = os.WriteFile(configPath, []byte("{ invalid json "), 0644)
	if err != nil {
		t.Fatal(err)
	}

	manager, err := NewConfigManagerAtPath(configPath)
	if err != nil {
		t.Fatalf("NewConfigManagerAtPath failed on invalid JSON: %v", err)
	}
	// Should fallback to default on load error (ignoring corruption for now, but at least not crashing)
	if manager.Get().PollingInterval != 1000 {
		t.Error("Expected default polling interval after loading invalid JSON")
	}

	// Test 2: Directory creation failure (e.g. file exists where dir should be)
	blockedDir := filepath.Join(tempDir, "blocked")
	os.WriteFile(blockedDir, []byte("i am a file"), 0644)
	_, err = NewConfigManagerAtPath(filepath.Join(blockedDir, "config.json"))
	if err == nil {
		t.Error("Expected error when config directory cannot be created")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.PollingInterval != 1000 {
		t.Errorf("expected 1000, got %d", cfg.PollingInterval)
	}
	if cfg.StoragePath == "" {
		t.Error("expected non-empty storage path")
	}
}
