// Chrome URL Tracker is a macOS menu bar application that monitors Google Chrome tabs
// and provides a seamless UI for saving and whitelisting URLs.
// It uses a concurrent architecture with a background monitor and a foreground menu loop.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"chrome-url-tracker/config"
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
	modeFlag := flag.String("mode", "", "Run in specific mode (search, settings)")
	flag.Parse()

	cfgManager, err := config.NewConfigManager()
	if err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}
	cfg := cfgManager.Get()

	store, err := storage.NewStore(cfg.StoragePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// TODO: do we need this?
	// This is a bit of a hack to get the UI to start in the background
	// and handle the IPC messages.
	if *modeFlag == "search" {
		runSearchMode(store)
		// Give UI some time to start before exiting Go if it was one-shot,
		// but since UI is a separate process we can just wait or stay alive.
		// For search mode, we stay alive to handle IPC.
		startIPCListener(store, cfgManager)
		select {} // Block
	}

	// TODO: do we need this?
	// This is a bit of a hack to get the UI to start in the background
	// and handle the IPC messages.
	if *modeFlag == "settings" {
		ui.ShowSettings(cfgManager.Get())
		startIPCListener(store, cfgManager)
		select {} // Block
	}

	// Default: Run as Menu Bar App with background monitor
	fmt.Printf("Starting Chrome URL Tracker (interval: %dms)...\n", cfg.PollingInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background IPC background loop
	startIPCListener(store, cfgManager)

	// Start monitor in background
	go runMonitorMode(ctx, store, cfgManager)

	// Start menu bar (blocks until exit)
	ui.StartMenu(ui.MenuHandlers{
		OnWhitelist: func() {
			// Try to get active tab to provide modern context-aware whitelisting
			tab, err := monitor.GetActiveTab()
			if err != nil || tab.URL == "" {
				ui.ShowDialog("Error", "An active browser tab (Chrome/Safari) is required to be whitelisted, please switch to the desired tab and try again.")
				return
			}

			u, err := url.Parse(tab.URL)
			var domain string
			if err == nil && u.Host != "" {
				// Remove www. for cleaner whitelist entry if preferred, but we'll stick to exact
				domain = u.Host
			} else {
				domain = tab.URL // Fallback
			}

			err = store.AddExcludedDomain(domain)
			if err != nil {
				ui.ShowNotification("Error", fmt.Sprintf("Failed to whitelist: %v", err))
			} else {
				ui.ShowNotification("Chrome Tracker", "Whitelisted: "+domain)
			}
		},
		OnManageWhitelist: func() {
			items := store.GetExcludedDomains()
			entries := store.GetEntries()
			ui.ShowDashboard("whitelist", items, entries)
		},
		OnPreferences: func() {
			ui.ShowSettings(cfgManager.Get())
		},
		OnManageBookmarks: func() { // Renamed from OnSearch
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
func startIPCListener(store *storage.Store, cfgManager *config.ConfigManager) {
	go func() {
		for {
			if msg, ok := ui.ConsumeIPCResult(); ok {
				handleIPCMessage(msg, store, cfgManager)
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func handleIPCMessage(msg string, store *storage.Store, cfgManager *config.ConfigManager) {
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
	case "SAVE_CONFIG":
		log.Printf("IPC: Handling SAVE_CONFIG")
		// Expect value to be integer (milliseconds)
		var interval int
		if _, err := fmt.Sscanf(value, "%d", &interval); err == nil {
			err = cfgManager.SetInterval(interval)
			if err != nil {
				ui.ShowNotification("Error", fmt.Sprintf("Failed to save config: %v", err))
			} else {
				ui.ShowNotification("Success", "Settings saved. Restart app to apply polling interval changes.")
			}
		}
	case "IMPORT_BOOKMARKS":
		log.Printf("IPC: Handling IMPORT_BOOKMARKS for %s", value)
		if value != "" {
			err := store.ImportBookmarks(value)
			if err != nil {
				ui.ShowNotification("Error", fmt.Sprintf("Failed to import bookmarks: %v", err))
			} else {
				ui.ShowNotification("Success", "Bookmarks imported successfully")
				ui.ShowDashboard("search", store.GetExcludedDomains(), store.GetEntries())
			}
		}
	case "EXPORT_BOOKMARKS":
		log.Printf("IPC: Handling EXPORT_BOOKMARKS to %s", value)
		if value != "" {
			err := store.ExportBookmarks(value)
			if err != nil {
				ui.ShowNotification("Error", fmt.Sprintf("Failed to export bookmarks: %v", err))
			} else {
				ui.ShowNotification("Success", "Bookmarks exported successfully")
			}
		}
	case "IMPORT_JSON":
		log.Printf("IPC: Handling IMPORT_JSON from %s", value)
		if value != "" {
			err := store.ImportNativeJSON(value)
			if err != nil {
				ui.ShowNotification("Error", fmt.Sprintf("Failed to import JSON backup: %v", err))
			} else {
				ui.ShowNotification("Success", "Native backup imported successfully")
				ui.ShowDashboard("search", store.GetExcludedDomains(), store.GetEntries())
			}
		}
	case "EXPORT_JSON":
		log.Printf("IPC: Handling EXPORT_JSON to %s", value)
		if value != "" {
			err := store.ExportNativeJSON(value)
			if err != nil {
				ui.ShowNotification("Error", fmt.Sprintf("Failed to export JSON backup: %v", err))
			} else {
				ui.ShowNotification("Success", "Native backup exported successfully")
			}
		}
	case "DELETE_ENTRY":
		log.Printf("IPC: Handling DELETE_ENTRY for %s", value)
		if value != "" {
			if ui.ShowConfirm("Confirm Removal", fmt.Sprintf("Remove %s from bookmarks?", value)) {
				err := store.RemoveEntry(value)
				if err != nil {
					ui.ShowNotification("Error", fmt.Sprintf("Failed to remove bookmark: %v", err))
				} else {
					ui.ShowNotification("Success", fmt.Sprintf("Removed bookmark: %s", value))
					// Refresh dashboard after deletion
					ui.ShowDashboard("search", store.GetExcludedDomains(), store.GetEntries())
				}
			}
		}
	case "UPDATE_ENTRY":
		log.Printf("IPC: Handling UPDATE_ENTRY for %s", value)
		if value != "" {
			var updatedEntry storage.Entry
			if err := json.Unmarshal([]byte(value), &updatedEntry); err == nil {
				err := store.UpdateEntry(updatedEntry)
				if err != nil {
					ui.ShowNotification("Error", fmt.Sprintf("Failed to update bookmark: %v", err))
				} else {
					ui.ShowNotification("Success", fmt.Sprintf("Updated bookmark: %s", updatedEntry.URL))
					// Refresh dashboard after update
					ui.ShowDashboard("search", store.GetExcludedDomains(), store.GetEntries())
				}
			} else {
				log.Printf("Error unmarshalling UPDATE_ENTRY payload: %v", err)
				ui.ShowNotification("Error", fmt.Sprintf("Failed to parse update data: %v", err))
			}
		}
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
func runMonitorMode(ctx context.Context, store *storage.Store, cfgManager *config.ConfigManager) {
	seenUrls := make(map[string]bool)

	interval := time.Duration(cfgManager.Get().PollingInterval) * time.Millisecond
	m := monitor.New(interval, func(tab monitor.TabInfo) bool {
		log.Printf("Monitor: Callback triggered for %s (Browser: %s)", tab.URL, tab.Browser)
		if atomic.LoadInt32(&isPaused) == 1 {
			log.Printf("Monitor: Skipping %s (monitoring paused)", tab.URL)
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
			log.Printf("Monitor: Skipping %s (whitelisted)", tab.URL)
			return true
		}

		if seenUrls[tab.URL] || store.EntryExists(tab.URL) {
			// Silently skip if already seen or exists to avoid spamming logs
			return true
		}

		log.Printf("Monitor: New URL detected in %s: %s", tab.Browser, tab.URL)

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
				log.Printf("Error: Failed to save URL %s: %v", tab.URL, err)
			} else {
				ui.ShowNotification("Chrome Tracker", "Saved: "+tab.Title)
				seenUrls[tab.URL] = true
				log.Printf("Monitor: Successfully saved %s", tab.URL)
			}
		} else {
			// Mark as seen anyway to avoid re-prompting immediately in this session
			seenUrls[tab.URL] = true
			log.Printf("Monitor: User skipped saving %s", tab.URL)
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
