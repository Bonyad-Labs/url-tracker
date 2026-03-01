// Package monitor provides functionality to track browser state.
// It currently supports Google Chrome and Safari on macOS via AppleScript polling.
package monitor

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// TabInfo holds the metadata for a single browser tab.
type TabInfo struct {
	URL     string
	Title   string
	Browser string // Which browser provided this info
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
	ticker := time.Tick(m.interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			info, err := GetActiveTab()
			if err != nil {
				// Log errors that aren't just "no supported browser" to help debugging
				if !strings.Contains(err.Error(), "no supported browser") {
					log.Printf("Monitor Error: %v", err)
				}
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
		try
    tell application "System Events"
        -- Get the name of the application currently in the foreground
        set frontApp to name of first application process whose frontmost is true
    end tell

    log "DEBUG: The frontmost app is: [" & frontApp & "]"

    -- Check if that application is Safari
    if frontApp is "Safari" or frontApp is "Safari Technology Preview" then
		log "DEBUG: Inside Frontmost app is Safari"
        tell application frontApp
			log "DEBUG: Inside tell application frontApp"
			set winCount to (count of windows) as integer
			log "DEBUG: Confirmed winCount as integer: " & winCount
            if winCount > 0 then
				try
					-- getting the title as below works, but getting the URL doesn't
					-- so we use the tell application "Safari" block to get both the URL and the title
					-- which seems to be working fine
					-- set currentTitle to name of document 1
					-- set currentURL to URL of document 1
					tell application "Safari"
						set currentTitle to name of document 1
						set currentURL to URL of document 1
					end tell
					if currentURL is not "missing value" and currentURL is not missing value then
						return "Safari" & "|||" & currentURL & "|||" & currentTitle
					end if
				on error err
					log "DEBUG: Error fetching doc properties: " & err
				end try
            end if
        end tell
	else if frontApp is "Google Chrome" or frontApp is "Google Chrome Beta" or frontApp is "Google Chrome Canary" then
		log "DEBUG: Inside Frontmost app is Google Chrome"
		tell application "Google Chrome"
			set winCount to (count of windows) as integer
			log "DEBUG: Confirmed winCount as integer: " & winCount
			if winCount > 0 then
				try
					set currentTitle to name of active tab of front window
					set currentURL to URL of active tab of front window
					if currentURL is not "missing value" and currentURL is not missing value then
						return "Google Chrome" & "|||" & currentURL & "|||" & currentTitle
					end if
				on error err
					log "DEBUG: Error fetching doc properties: " & err
				end try
			end if
		end tell
    end if
    
    -- Return empty strings (separated by your delimiter) if not Safari
    return ""
on error
	log "DEBUG: Error getting the frontmost app: " & err
    return ""
end try`

	cmd := exec.Command("osascript", "-")
	cmd.Stdin = strings.NewReader(script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// 1. Print Logs (Stderr)
	if stderr.Len() > 0 {
		log.Printf("AppleScript Logs:\n%s\n", stderr.String())
	}

	if err != nil {
		log.Printf("Execution Error: %v\n", err)
		return TabInfo{}, fmt.Errorf("osascript error: %w", err)
	}

	// 2. Process Result (Stdout)
	result := strings.TrimSpace(stdout.String())
	if result == "|||" || result == "" {
		log.Println("No Safari tab found in foreground.")
	} else {
		parts := strings.Split(result, "|||")
		log.Printf("URL: %s\nTitle: %s\n", parts[0], parts[1])
	}

	return parseActiveTab(result)
}

func parseActiveTab(output string) (TabInfo, error) {
	result := strings.TrimSpace(output)
	if result == "" || result == "NONE" || strings.Contains(result, "missing value") {
		return TabInfo{}, fmt.Errorf("no supported browser running or no tabs open (result: %s)", result)
	}

	if strings.HasPrefix(result, "ERROR:") {
		return TabInfo{}, fmt.Errorf("applescript error: %s", strings.TrimPrefix(result, "ERROR:"))
	}

	if strings.HasPrefix(result, "APP:") {
		// Log the frontmost app to help debugging if it's not a browser we track
		appName := strings.TrimPrefix(result, "APP:")
		return TabInfo{}, fmt.Errorf("no supported browser running (frontmost app: %s)", appName)
	}

	parts := strings.Split(result, "|||")
	if len(parts) < 3 {
		return TabInfo{}, fmt.Errorf("invalid script output: %s (expected 3 parts)", result)
	}

	return TabInfo{
		Browser: parts[0],
		URL:     parts[1],
		Title:   parts[2],
	}, nil
}
