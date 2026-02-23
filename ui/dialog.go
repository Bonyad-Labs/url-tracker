// Package ui provides native macOS user interface components.
// It uses a hybrid approach: lightweight dialogs via AppleScript (osascript)
// and complex management windows via compiled SwiftUI binaries.
package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// uiMu ensures that only one AppleScript dialog is visible at a time.
// This prevents multiple dialogs from overlapping and causing diagnostic errors.
var uiMu sync.Mutex

func getCmdPath() string {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	cmdPath := userHomeDir + "/usr/local/bin/whitelist-manager"
	if _, err := os.Stat("./whitelist-manager"); err == nil {
		cmdPath = "./whitelist-manager"
	}

	return cmdPath
}

// ShowWhitelistManager displays the native SwiftUI whitelist manager window.
func ShowWhitelistManager(items interface{}) (selected string, ok bool) {
	data, err := json.Marshal(items)
	if err != nil {
		return "", false
	}

	cmdPath := getCmdPath()
	if cmdPath == "" {
		return "", false
	}

	uiMu.Lock()
	defer uiMu.Unlock()

	cmd := exec.Command(cmdPath, "--mode", "whitelist", "--data", string(data))
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return "", false
	}

	return output, true
}

// ShowSearchManager displays the native SwiftUI search manager window.
// It returns the action (open/copy) and the associated value (URL).
func ShowSearchManager(entries interface{}) (action string, value string, ok bool) {
	data, err := json.Marshal(entries)
	if err != nil {
		return "", "", false
	}

	cmdPath := getCmdPath()
	if cmdPath == "" {
		return "", "", false
	}

	uiMu.Lock()
	defer uiMu.Unlock()

	cmd := exec.Command(cmdPath, "--mode", "search", "--data", string(data))
	out, err := cmd.Output()
	if err != nil {
		return "", "", false
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return "", "", false
	}

	parts := strings.Split(output, "|")
	if len(parts) == 2 {
		return strings.ToLower(parts[0]), parts[1], true
	}

	return "", "", false
}

// ShowAddWhitelistDialog displays a native SwiftUI dialog to choose between
// whitelisting the domain or the specific URL.
func ShowAddWhitelistDialog(url, title string) (selection string, ok bool) {
	cmdPath := getCmdPath()
	if cmdPath == "" {
		return "", false
	}

	uiMu.Lock()
	defer uiMu.Unlock()

	cmd := exec.Command(cmdPath, "--mode", "add", "--url", url, "--title", title)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return "", false
	}

	return output, true
}

// ShowSaveDialog displays a native SwiftUI form to capture URL metadata.
// It returns the metadata fields and flags for save/whitelist actions.
func ShowSaveDialog(url, title string) (description string, tags []string, category string, saved bool, whitelist bool) {
	cmdPath := getCmdPath()
	if cmdPath == "" {
		return "", nil, "", false, false
	}

	uiMu.Lock()
	defer uiMu.Unlock()

	cmd := exec.Command(cmdPath, "--mode", "save", "--url", url, "--title", title)
	out, err := cmd.Output()
	if err != nil {
		return "", nil, "", false, false
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return "", nil, "", false, false
	}

	// Parse JSON output
	var res struct {
		Action      string `json:"action"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Tags        string `json:"tags"`
	}

	if err := json.Unmarshal([]byte(output), &res); err != nil {
		// Fallback for simple actions if JSON fails (though it shouldn't)
		if strings.Contains(output, "whitelist") {
			return "", nil, "", false, true
		}
		return "", nil, "", false, false
	}

	if res.Action == "whitelist" {
		return "", nil, "", false, true
	}

	if res.Action == "save" {
		var tagList []string
		if res.Tags != "" {
			parts := strings.Split(res.Tags, ",")
			for _, p := range parts {
				tagList = append(tagList, strings.TrimSpace(p))
			}
		}
		return res.Description, tagList, res.Category, true, false
	}

	return "", nil, "", false, false
}

// ShowNotification displays a native macOS system notification.
func ShowNotification(title, message string) {
	script := fmt.Sprintf(`display notification %s with title %s`, quoteForAppleScript(message), quoteForAppleScript(title))
	runAppleScript(script)
}

// ShowConfirm displays a standard OK/Cancel confirmation dialog.
func ShowConfirm(title, message string) bool {
	script := fmt.Sprintf(`display dialog %s with title %s buttons {"Cancel", "OK"} default button "OK"`, quoteForAppleScript(message), quoteForAppleScript(title))
	_, err := runAppleScript(script)
	return err == nil
}

// ShowSearchResults displays a selection list of search matches.
func ShowSearchResults(results []string) (int, bool) {
	if len(results) == 0 {
		ShowNotification("Chrome URL Tracker", "No results found")
		return -1, false
	}

	listStr := "{"
	for i, r := range results {
		listStr += quoteForAppleScript(r)
		if i < len(results)-1 {
			listStr += ", "
		}
	}
	listStr += "}"

	script := fmt.Sprintf(`choose from list %s with title "Chrome URL Tracker" with prompt "Select an entry to view details:"`, listStr)
	output, err := runAppleScript(script)
	output = strings.TrimSpace(output)
	if err != nil || output == "false" || output == "" {
		return -1, false
	}

	selected := output
	for i, r := range results {
		if r == selected {
			return i, true
		}
	}
	return -1, false
}

// ShowEntryDetails displays the full metadata of a saved URL and offers actions (Copy/Open).
func ShowEntryDetails(details string) (string, bool) {
	script := fmt.Sprintf(`display dialog %s with title "Entry Details" buttons {"Back", "Copy URL", "Open in Chrome"} default button "Open in Chrome"`, quoteForAppleScript(details))
	output, err := runAppleScript(script)
	if err != nil {
		return "", false
	}

	if strings.Contains(output, "Copy URL") {
		return "copy", true
	}
	if strings.Contains(output, "Open in Chrome") {
		return "open", true
	}
	if strings.Contains(output, "Back") {
		return "back", true
	}

	return "", false
}

// ShowInputDialog captures a single line of text from the user with custom buttons.
// Returns the input text, the button clicked, and a success flag.
func ShowInputDialog(title, prompt, defaultAnswer string, buttons []string) (string, string, bool) {
	btnStr := ""
	if len(buttons) > 0 {
		btnStr = "buttons {"
		for i, b := range buttons {
			btnStr += quoteForAppleScript(b)
			if i < len(buttons)-1 {
				btnStr += ", "
			}
		}
		btnStr += "} default button " + quoteForAppleScript(buttons[len(buttons)-1])
	}

	script := fmt.Sprintf(`display dialog %s with title %s default answer %s %s`,
		quoteForAppleScript(prompt), quoteForAppleScript(title), quoteForAppleScript(defaultAnswer), btnStr)

	output, err := runAppleScript(script)
	if err != nil {
		return "", "", false
	}

	text, button := extractTextAndButtonFromDialog(output)
	return text, button, true
}

func runAppleScript(script string) (string, error) {
	uiMu.Lock()
	defer uiMu.Unlock()

	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func quoteForAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", `\n`) // AppleScript needs actual newline characters in some contexts but usually \n works in strings
	return "\"" + s + "\""
}

func extractTextAndButtonFromDialog(output string) (text string, button string) {
	// Dialog output format: "button returned:OK, text returned:user input"
	btnParts := strings.Split(output, "button returned:")
	textParts := strings.Split(output, "text returned:")

	if len(btnParts) > 1 {
		button = strings.Split(btnParts[1], ",")[0]
	}

	if len(textParts) > 1 {
		text = strings.TrimSpace(textParts[1])
	}

	return text, button
}
