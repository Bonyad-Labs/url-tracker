// Package monitor provides functionality to track browser state.
// It currently supports Google Chrome on macOS via AppleScript polling.
package monitor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// TabInfo holds the metadata for a single browser tab.
type TabInfo struct {
	URL   string
	Title string
}

// ChromeMonitor polls Google Chrome for changes to the active tab.
type ChromeMonitor struct {
	interval time.Duration // How often to poll Chrome
	onChange func(TabInfo) // Callback triggered when a new URL is detected
	lastURL  string        // The last seen URL to detect changes
}

// New initializes a new ChromeMonitor with the specified poll interval.
func New(interval time.Duration, onChange func(TabInfo)) *ChromeMonitor {
	return &ChromeMonitor{
		interval: interval,
		onChange: onChange,
	}
}

// Start begins the polling loop. It blocks until the context is cancelled.
// Each poll executes an AppleScript to query Chrome's state.
func (m *ChromeMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := GetActiveTab()
			if err != nil {
				// Silently skip if Chrome is not running or other errors
				continue
			}

			if info.URL != "" && info.URL != m.lastURL {
				m.lastURL = info.URL
				m.onChange(info)
			}
		}
	}
}

// GetActiveTab executes an AppleScript to query Chrome's active tab URL and title.
func GetActiveTab() (TabInfo, error) {
	// AppleScript to get active tab's URL and title from Google Chrome
	script := `
		if application "Google Chrome" is running then
			tell application "Google Chrome"
				if (count of windows) > 0 then
					tell active tab of front window
						return URL & "|||" & title
					end tell
				end if
			end tell
		end if
		return ""`

	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return TabInfo{}, err
	}

	return parseActiveTab(string(out))
}

func parseActiveTab(output string) (TabInfo, error) {
	result := strings.TrimSpace(output)
	if result == "" {
		return TabInfo{}, fmt.Errorf("chrome not running or no tabs open")
	}

	parts := strings.Split(result, "|||")
	if len(parts) < 2 {
		return TabInfo{}, fmt.Errorf("invalid script output")
	}

	return TabInfo{
		URL:   parts[0],
		Title: parts[1],
	}, nil
}
