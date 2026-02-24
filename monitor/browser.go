// Package monitor provides functionality to track browser state.
// It currently supports Google Chrome and Safari on macOS via AppleScript polling.
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

// BrowserMonitor polls the frontmost browser for changes to the active tab.
type BrowserMonitor struct {
	interval time.Duration      // How often to poll the browser
	onChange func(TabInfo) bool // Callback triggered when a new URL is detected. Return true to update lastURL.
	lastURL  string             // The last seen URL to detect changes
}

// New initializes a new BrowserMonitor with the specified poll interval.
func New(interval time.Duration, onChange func(TabInfo) bool) *BrowserMonitor {
	return &BrowserMonitor{
		interval: interval,
		onChange: onChange,
	}
}

// Start begins the polling loop. It blocks until the context is cancelled.
// Each poll executes an AppleScript to query the browser's state.
func (m *BrowserMonitor) Start(ctx context.Context) {
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
				if m.onChange(info) {
					m.lastURL = info.URL
				}
			}
		}
	}
}

// GetActiveTab executes an AppleScript to query the frontmost browser's active tab URL and title.
func GetActiveTab() (TabInfo, error) {
	// AppleScript to get active tab's URL and title based on the frontmost app
	script := `
		tell application "System Events"
			set frontApp to name of first application process whose frontmost is true
		end tell
		
		if frontApp is "Google Chrome" then
			tell application "Google Chrome"
				if (count of windows) > 0 then
					tell active tab of front window
						return URL & "|||" & title
					end tell
				end if
			end tell
		else if frontApp is "Safari" then
			tell application "Safari"
				if (count of windows) > 0 then
					tell current tab of front window
						return URL & "|||" & name
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
