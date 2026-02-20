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
)

// isSearching is an atomic flag to prevent multiple search sessions from overlapping.
var isSearching int32

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

	// Start monitor in background
	go runMonitorMode(ctx, store, time.Duration(*intervalFlag)*time.Millisecond)

	// Start menu bar (blocks until exit)
	ui.StartMenu(ui.MenuHandlers{
		OnWhitelist: func() {
			// Try to get active tab to provide modern context-aware whitelisting
			tab, err := monitor.GetActiveTab()
			if err == nil {
				selection, ok := ui.ShowAddWhitelistDialog(tab.URL, tab.Title)
				if ok {
					err := store.AddExcludedDomain(selection)
					if err != nil {
						ui.ShowNotification("Error", fmt.Sprintf("Failed to whitelist: %v", err))
					} else {
						ui.ShowNotification("Success", fmt.Sprintf("Whitelisted: %s", selection))
					}
				}
				return
			}

			// Fallback: Manual input if no active tab found
			domain, _, ok := ui.ShowInputDialog("Add to Whitelist", "Enter domain to exclude (e.g. youtube.com):", "", []string{"Cancel", "OK"})
			if ok && domain != "" {
				err := store.AddExcludedDomain(domain)
				if err != nil {
					ui.ShowNotification("Error", fmt.Sprintf("Failed to add whitelist: %v", err))
				} else {
					ui.ShowNotification("Success", fmt.Sprintf("Whitelisted: %s", domain))
				}
			}
		},
		// Handle "Manage Whitelist" action from the menu bar
		OnManageWhitelist: func() {
			items := store.GetExcludedDomains()
			if len(items) == 0 {
				ui.ShowNotification("Chrome Tracker", "Whitelist is empty")
				return
			}

			selected, ok := ui.ShowWhitelistManager(items)
			if ok && selected != "" {
				if ui.ShowConfirm("Confirm Removal", fmt.Sprintf("Remove %s from whitelist?", selected)) {
					err := store.RemoveExcludedDomain(selected)
					if err != nil {
						ui.ShowNotification("Error", fmt.Sprintf("Failed to remove: %v", err))
					} else {
						ui.ShowNotification("Success", fmt.Sprintf("Removed from whitelist: %s", selected))
					}
				}
			}
		},
		OnSearch: func() {
			if atomic.CompareAndSwapInt32(&isSearching, 0, 1) {
				go func() {
					runSearchMode(store)
					atomic.StoreInt32(&isSearching, 0)
				}()
			} else {
				ui.ShowNotification("Chrome Tracker", "Search is already active")
			}
		},
		OnQuit: func() {
			cancel()
			os.Exit(0)
		},
	})
}

// runMonitorMode executes the polling loop and coordinates detection logic.
// It is designed to run as a long-lived goroutine.
func runMonitorMode(ctx context.Context, store *storage.Store, interval time.Duration) {
	seenUrls := make(map[string]bool)

	m := monitor.New(interval, func(tab monitor.TabInfo) {
		if store.IsExcluded(tab.URL) {
			return
		}

		if seenUrls[tab.URL] || store.EntryExists(tab.URL) {
			return
		}

		// Show native modern save dialog
		desc, tags, cat, saved, whitelist := ui.ShowSaveDialog(tab.URL, tab.Title)
		if whitelist {
			selection, ok := ui.ShowAddWhitelistDialog(tab.URL, tab.Title)
			if ok {
				err := store.AddExcludedDomain(selection)
				if err != nil {
					ui.ShowNotification("Error", fmt.Sprintf("Failed to whitelist: %v", err))
				} else {
					ui.ShowNotification("Success", fmt.Sprintf("Whitelisted: %s", selection))
					seenUrls[tab.URL] = true
				}
			}
			return
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
				ui.ShowNotification("Success", "URL saved successfully")
				seenUrls[tab.URL] = true
			}
		} else {
			// Mark as seen anyway to avoid re-prompting immediately in this session
			seenUrls[tab.URL] = true
		}
	})

	m.Start(ctx)
}

// runSearchMode executes the interactive search interface using the native SwiftUI manager.
func runSearchMode(store *storage.Store) {
	entries := store.GetEntries()
	if len(entries) == 0 {
		ui.ShowNotification("Search Mode", "No saved URLs found")
		return
	}

	action, value, ok := ui.ShowSearchManager(entries)
	if !ok {
		return
	}

	switch action {
	case "open":
		openURLInChrome(value)
		ui.ShowNotification("Success", "Opening in Chrome")
	case "copy":
		copyToClipboard(value)
		ui.ShowNotification("Success", "URL copied to clipboard")
	}
}

func openURLInChrome(url string) {
	exec.Command("open", "-a", "Google Chrome", url).Run()
}

func copyToClipboard(text string) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	cmd.Run()
}
