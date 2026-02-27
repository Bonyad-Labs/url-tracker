// Chrome URL Tracker is a macOS menu bar application that monitors Google Chrome tabs
// and provides a seamless UI for saving and whitelisting URLs.
// It uses a concurrent architecture with a background monitor and a foreground menu loop.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"chrome-url-tracker/monitor"
	"chrome-url-tracker/storage"
	"chrome-url-tracker/ui"

	"github.com/getlantern/systray"
)

// isSearching is an atomic flag to prevent multiple search sessions from overlapping.
// isPaused is an atomic flag to control the monitoring state (0 = active, 1 = paused).
var (
	isSearching int32 = 0
	isPaused    int32 = 0
)

// main initializes the application, storage, and starts both the menu and monitoring loops.
func main() {
	searchFlag := flag.Bool("search", false, "Run in search mode one-shot")
	intervalFlag := flag.Int("interval", 1000, "Polling interval in milliseconds")
	storageFlag := flag.String("storage", "", "Path to SQLite database (default: ~/Library/Application Support/chrome-url-tracker/chrome-urls.db)")
	flag.Parse()

	store, err := storage.NewStore(*storageFlag)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	if *searchFlag {
		runSearchMode(store)
		return
	}

	// Default: Run as Menu Bar App with background monitor
	fmt.Printf("Starting Chrome URL Tracker (interval: %dms)...\n", *intervalFlag)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background IPC background loop
	startIPCListener(store)

	// Start monitor in background
	go runMonitorMode(ctx, store, time.Duration(*intervalFlag)*time.Millisecond)

	// Start menu bar (blocks until exit)
	ui.StartMenu(ui.MenuHandlers{
		OnWhitelist: func() {
			// Try to get active tab to provide modern context-aware whitelisting
			tab, err := monitor.GetActiveTab()
			tabURL := "localhost"
			tabTitle := "localhost"
			if err == nil {
				tabURL = tab.URL
				tabTitle = tab.Title
			}
			ui.ShowAddWhitelistDialog(tabURL, tabTitle)
		},
		OnManageWhitelist: func() {
			items := store.GetExcludedDomains()
			entries := store.GetEntries()
			ui.ShowDashboard("whitelist", items, entries)
		},
		OnSearch: func() {
			go runSearchMode(store)
		},
		OnTogglePause: func(item *systray.MenuItem) {
			// Toggle between 0 (active) and 1 (paused)
			if atomic.CompareAndSwapInt32(&isPaused, 0, 1) {
				item.SetTitle("Resume Monitoring")
				ui.ShowNotification("Chrome Tracker", "Monitoring Paused")
			} else {
				atomic.StoreInt32(&isPaused, 0)
				item.SetTitle("Pause Monitoring")
				ui.ShowNotification("Chrome Tracker", "Monitoring Resumed")
			}
		},
		OnQuit: func() {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Println("Error finding home directory:", err)
				return
			}
			cmd := exec.Command("launchctl", "unload", homeDir+"/Library/LaunchAgents/com.user.chrome-url-tracker.plist")
			output, err := cmd.CombinedOutput()
			if err != nil {
				// Log error and command output for debugging
				log.Fatalf("Failed to unload launchctl job: %v\nOutput: %s", err, string(output))
			}
			os.Exit(0)
		},
	})
}

// startIPCListener runs in the background to handle commands returning from the persistent Swift app.
func startIPCListener(store *storage.Store) {
	go func() {
		for {
			if msg, ok := ui.ConsumeIPCResult(); ok {
				handleIPCMessage(msg, store)
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func handleIPCMessage(msg string, store *storage.Store) {
	parts := strings.SplitN(msg, "|", 2)
	if len(parts) == 0 {
		return
	}

	action := parts[0]
	value := ""
	if len(parts) > 1 {
		value = parts[1]
	}

	switch action {
	case "ADD_WHITELIST":
		if value != "" {
			err := store.AddExcludedDomain(value)
			if err != nil {
				ui.ShowNotification("Error", fmt.Sprintf("Failed to whitelist: %v", err))
			} else {
				ui.ShowNotification("Success", fmt.Sprintf("Whitelisted: %s", value))
				// Refresh the UI to reflect changes
				ui.ShowDashboard("whitelist", store.GetExcludedDomains(), store.GetEntries())
			}
		}
	case "DELETE_WHITELIST":
		if ui.ShowConfirm("Confirm Removal", fmt.Sprintf("Remove %s from whitelist?", value)) {
			err := store.RemoveExcludedDomain(value)
			if err != nil {
				ui.ShowNotification("Error", fmt.Sprintf("Failed to remove: %v", err))
			} else {
				ui.ShowNotification("Success", fmt.Sprintf("Removed from whitelist: %s", value))
				// Refresh the UI to reflect changes
				ui.ShowDashboard("whitelist", store.GetExcludedDomains(), store.GetEntries())
			}
		}
	case "OPEN":
		openURLInChrome(value)
		ui.ShowNotification("Success", "Opening in Chrome")
	case "COPY":
		copyToClipboard(value)
		ui.ShowNotification("Success", "URL copied to clipboard")
	}
}

// runMonitorMode executes the polling loop and coordinates detection logic.
// It is designed to run as a long-lived goroutine.
func runMonitorMode(ctx context.Context, store *storage.Store, interval time.Duration) {
	seenUrls := make(map[string]bool)

	m := monitor.New(interval, func(tab monitor.TabInfo) bool {
		if atomic.LoadInt32(&isPaused) == 1 {
			return false // Silently skip and don't update lastURL
		}

		// Strip URL of query parameters and fragment
		if idx := strings.Index(tab.URL, "?"); idx != -1 {
			tab.URL = tab.URL[:idx]
		}
		if idx := strings.Index(tab.URL, "#"); idx != -1 {
			tab.URL = tab.URL[:idx]
		}

		if store.IsExcluded(tab.URL) {
			return true
		}

		if seenUrls[tab.URL] || store.EntryExists(tab.URL) {
			return true
		}

		// Show native modern save dialog
		desc, tags, cat, saved, whitelist := ui.ShowSaveDialog(tab.URL, tab.Title)
		if whitelist {
			ui.ShowAddWhitelistDialog(tab.URL, tab.Title)
			seenUrls[tab.URL] = true
			return true
		}

		if saved {
			entry := storage.Entry{
				URL:         tab.URL,
				Title:       tab.Title,
				Description: desc,
				Tags:        tags,
				Category:    cat,
			}
			err := store.AddEntry(entry)
			if err != nil {
				ui.ShowNotification("Error", fmt.Sprintf("Failed to save URL: %v", err))
			} else {
				ui.ShowNotification("Chrome Tracker", "Saved: "+tab.Title)
				seenUrls[tab.URL] = true
			}
		} else {
			// Mark as seen anyway to avoid re-prompting immediately in this session
			seenUrls[tab.URL] = true
		}
		return true
	})

	m.Start(ctx)
}

// runSearchMode executes the interactive search interface using the native SwiftUI manager.
func runSearchMode(store *storage.Store) {
	entries := store.GetEntries()
	items := store.GetExcludedDomains()
	ui.ShowDashboard("search", items, entries)
}

func openURLInChrome(url string) {
	exec.Command("open", "-a", "Google Chrome", url).Run()
}

func copyToClipboard(text string) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	cmd.Run()
}
