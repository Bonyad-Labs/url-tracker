package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// uiMu ensures that only one AppleScript dialog is visible at a time.
// This prevents multiple dialogs from overlapping and causing diagnostic errors.
var uiMu sync.Mutex

// ShowForm orchestrates a sequence of AppleScript dialogs to capture URL metadata.
// It bypasses the 3-button limit of AppleScript by chaining calls.
// Returns description, tags, category, and flags for save/whitelist actions.
func ShowForm(title, url string) (description string, tags []string, category string, saved bool, whitelist bool) {
	// Dialog 1: Description + Whitelist button
	descResult, button, ok := ShowInputDialog("Chrome URL Tracker", fmt.Sprintf("URL: %s\n\nEnter Description:", url), "", []string{"Cancel", "Whitelist", "OK"})
	if !ok {
		return "", nil, "", false, false
	}
	if button == "Whitelist" {
		return "", nil, "", false, true
	}
	description = descResult

	// Dialog 2: Category
	catResult, _, ok := ShowInputDialog("Chrome URL Tracker", "Enter Category (e.g., Research, Social, Work):", "Research", []string{"Cancel", "OK"})
	if !ok {
		return description, nil, "", false, false
	}
	category = catResult

	// Dialog 4: Tags
	tagsResult, _, ok := ShowInputDialog("Chrome URL Tracker", "Enter Tags (comma separated):", "", []string{"Cancel", "OK"})
	if !ok {
		return description, nil, category, false, false
	}
	if tagsResult != "" {
		parts := strings.Split(tagsResult, ",")
		for _, p := range parts {
			tags = append(tags, strings.TrimSpace(p))
		}
	}

	return description, tags, category, true, false
}

// ShowWhitelistOptions prompts the user to choose between whitelisting a domain or a full URL.
func ShowWhitelistOptions(url string) (choice string, ok bool) {
	prompt := fmt.Sprintf("Choose whitelisting option for:\n%s", url)
	script := fmt.Sprintf(`display dialog %s with title "Whitelist Options" buttons {"Cancel", "Whitelist URL", "Whitelist Domain"} default button "Whitelist Domain"`, quoteForAppleScript(prompt))
	output, err := runAppleScript(script)
	if err != nil {
		return "", false
	}

	if strings.Contains(output, "button returned:Whitelist Domain") {
		return "domain", true
	}
	if strings.Contains(output, "button returned:Whitelist URL") {
		return "url", true
	}
	return "", false
}

// ShowWhitelistManager displays the list of currently whitelisted items for removal.
func ShowWhitelistManager(items []string) (selected string, ok bool) {
	if len(items) == 0 {
		ShowNotification("Chrome URL Tracker", "Whitelist is empty")
		return "", false
	}

	listStr := "{"
	for i, r := range items {
		listStr += quoteForAppleScript(r)
		if i < len(items)-1 {
			listStr += ", "
		}
	}
	listStr += "}"

	script := fmt.Sprintf(`choose from list %s with title "Whitelist Manager" with prompt "Select an item to remove from whitelist:" OK button name "Remove" cancel button name "Cancel"`, listStr)
	output, err := runAppleScript(script)
	if err != nil || output == "false" || strings.TrimSpace(output) == "" {
		return "", false
	}

	return strings.TrimSpace(output), true
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
	if err != nil || output == "false" {
		return -1, false
	}

	selected := strings.TrimSpace(output)
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
