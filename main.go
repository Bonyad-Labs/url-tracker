// Chrome URL Tracker is a macOS menu bar application that monitors Google Chrome tabs
// and provides a seamless UI for saving and whitelisting URLs.
// It uses a concurrent architecture with a background monitor and a foreground menu loop.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
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
	storageFlag := flag.String("storage", "~/Documents/chrome-urls.json", "Path to storage JSON file")
	flag.Parse()

	store, err := storage.NewStore(*storageFlag)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

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

		// Show sequential dialogs
		desc, tags, cat, saved, whitelist := ui.ShowForm("New URL Detected", tab.URL)
		if whitelist {
			choice, ok := ui.ShowWhitelistOptions(tab.URL)
			if ok {
				toExclude := tab.URL
				if choice == "domain" {
					u, err := url.Parse(tab.URL)
					if err == nil {
						toExclude = u.Host
					}
				}
				err := store.AddExcludedDomain(toExclude)
				if err != nil {
					ui.ShowNotification("Error", fmt.Sprintf("Failed to whitelist: %v", err))
				} else {
					ui.ShowNotification("Success", fmt.Sprintf("Whitelisted: %s", toExclude))
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

// runSearchMode executes the interactive search interface loop.
func runSearchMode(store *storage.Store) {
	for {
		query, _, ok := ui.ShowInputDialog("Search Saved URLs", "Enter search query:", "", []string{"Cancel", "Search"})
		if !ok {
			return
		}

		results := store.SearchEntries(query)
		if len(results) == 0 {
			ui.ShowNotification("Search Mode", "No matches found")
			continue
		}

		var displayStrings []string
		for _, r := range results {
			displayStrings = append(displayStrings, fmt.Sprintf("%s - %s", r.Title, r.URL))
		}

		idx, ok := ui.ShowSearchResults(displayStrings)
		if !ok {
			continue
		}

		selected := results[idx]
		for {
			details := fmt.Sprintf("Title: %s\nURL: %s\nCategory: %s\nTags: %s\n\nDescription:\n%s",
				selected.Title, selected.URL, selected.Category, strings.Join(selected.Tags, ", "), selected.Description)

			action, ok := ui.ShowEntryDetails(details)
			if !ok || action == "back" {
				break
			}

			if action == "copy" {
				copyToClipboard(selected.URL)
				ui.ShowNotification("Copy", "URL copied to clipboard")
			} else if action == "open" {
				openURLInChrome(selected.URL)
				return // Exit after opening
			}
		}
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
