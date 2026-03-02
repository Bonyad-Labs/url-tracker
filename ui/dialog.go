// Package ui provides native macOS user interface components.
// It uses a hybrid approach: lightweight dialogs via AppleScript (osascript)
// and complex management windows via compiled SwiftUI binaries.
package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// uiMu protects access to the shared UI process and channels
var uiMu sync.Mutex
var uiProcess *exec.Cmd
var uiStdin io.WriteCloser
var uiResultChan chan string

type ipcCommand struct {
	Mode          string      `json:"mode"`
	SearchData    interface{} `json:"searchData,omitempty"`
	WhitelistData interface{} `json:"whitelistData,omitempty"`
	ConfigData    interface{} `json:"configData,omitempty"`
	URL           string      `json:"url,omitempty"`
	Title         string      `json:"title,omitempty"`
}

func getCmdPath() string {
	execPath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(execPath)
		appCmdPath := filepath.Join(dir, "whitelist-manager")
		if _, err := os.Stat(appCmdPath); err == nil {
			return appCmdPath
		}
	}

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

// ensureUIProcess starts the SwiftUI application in the background if it isn't already running.
func ensureUIProcess() error {
	uiMu.Lock()
	defer uiMu.Unlock()

	if uiProcess != nil && uiProcess.ProcessState == nil {
		log.Printf("UI Process already running (PID: %d)", uiProcess.Process.Pid)
		return nil // Already running
	}

	cmdPath := getCmdPath()
	if cmdPath == "" {
		return fmt.Errorf("whitelist-manager not found")
	}

	log.Printf("Starting UI process: %s", cmdPath)
	uiProcess = exec.Command(cmdPath, "--mode", "dashboard")
	stdin, err := uiProcess.StdinPipe()
	if err != nil {
		return err
	}
	uiStdin = stdin

	stdout, err := uiProcess.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := uiProcess.StderrPipe()
	if err != nil {
		return err
	}

	uiResultChan = make(chan string, 10)

	if err := uiProcess.Start(); err != nil {
		return err
	}

	// Capture the local reference to the command so we can wait on it safely
	cmd := uiProcess
	// Background goroutine to read STDOUT from Swift
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text != "" {
				// Non-blocking send
				select {
				case uiResultChan <- text:
				default:
				}
			}
		}

		// Wait for the process to exit cleanly to avoid zombie processes
		err := cmd.Wait()
		log.Printf("UI Process exited. Error: %v", err)

		uiMu.Lock()
		if uiProcess == cmd {
			uiProcess = nil
		}
		uiMu.Unlock()
	}()

	// TODO: Eventually we need to handle the errors from the UI process better
	// For now, we just log them. but we should probably handle them in a more robust way
	// Background goroutine to read STDERR from Swift
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text != "" {
				log.Printf("Swift Stderr: %s", text)
			}
		}
	}()

	return nil
}

// sendIPCCommand sends a JSON command to the running Swift app.
func sendIPCCommand(cmd ipcCommand) error {
	if err := ensureUIProcess(); err != nil {
		return err
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	log.Printf("Sending IPC command: %.100s...", string(data))
	uiMu.Lock()
	defer uiMu.Unlock()
	_, err = uiStdin.Write(append(data, '\n'))
	return err
}

// ShowDashboard displays the native SwiftUI dashboard with both data models and config.
func ShowDashboard(mode string, whitelistItems interface{}, searchEntries interface{}, configData interface{}) (action string, value string, ok bool) {
	cmd := ipcCommand{
		Mode:          mode,
		WhitelistData: whitelistItems,
		SearchData:    searchEntries,
		ConfigData:    configData,
	}
	sendIPCCommand(cmd)
	return "", "", true
}

// ShowSettings displays the native SwiftUI settings pane.
func ShowSettings(configData interface{}) {
	cmd := ipcCommand{
		Mode:       "settings",
		ConfigData: configData,
	}
	sendIPCCommand(cmd)
}

// ShowAddWhitelistDialog displays the Add Whitelist dialog in the Unified UI Dashboard.
func ShowAddWhitelistDialog(url, title string) {
	cmd := ipcCommand{
		Mode:  "add",
		URL:   url,
		Title: title,
	}
	sendIPCCommand(cmd)
}

// ShowSaveDialog displays a native SwiftUI form to capture URL metadata.
func ShowSaveDialog(url, title string) (description string, tags []string, category string, saved bool, whitelist bool) {
	// For Save, we also launch synchronously as it interrupts monitor flow
	cmdPath := getCmdPath()
	if cmdPath == "" {
		return "", nil, "", false, false
	}

	cmd := exec.Command(cmdPath, "--mode", "save", "--url", url, "--title", title)
	out, err := cmd.Output()
	if err != nil {
		return "", nil, "", false, false
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return "", nil, "", false, false
	}

	parts := strings.SplitN(output, "|", 2)
	if len(parts) == 2 && parts[0] == "SAVE_ENTRY" {
		var res struct {
			Action      string `json:"action"`
			Description string `json:"description"`
			Category    string `json:"category"`
			Tags        string `json:"tags"`
		}
		if err := json.Unmarshal([]byte(parts[1]), &res); err != nil {
			return "", nil, "", false, false
		}

		var tagList []string
		if res.Tags != "" {
			tParts := strings.Split(res.Tags, ",")
			for _, p := range tParts {
				tagList = append(tagList, strings.TrimSpace(p))
			}
		}
		return res.Description, tagList, res.Category, true, false
	}

	if output == "ACTION_WHITELIST|" {
		return "", nil, "", false, true
	}

	return "", nil, "", false, false
}

// ConsumeIPCResult reads the next message from the Swift IPC channel, if any.
func ConsumeIPCResult() (string, bool) {
	select {
	case msg := <-uiResultChan:
		log.Printf("Received IPC message from Swift: %s", msg)
		return msg, true
	default:
		return "", false
	}
}

// ShowNotification displays a native macOS system notification.
func ShowNotification(title, message string) {
	script := fmt.Sprintf(`display notification %s with title %s`, quoteForAppleScript(message), quoteForAppleScript(title))
	// ShowNotification deliberately does NOT acquire uiMu so it can slide in without blocking,
	// allowing us to show warnings when the mutex is already locked.
	cmd := exec.Command("osascript", "-e", script)
	cmd.Run()
}

// ShowDialog displays a native macOS system notification.
func ShowDialog(title, message string) {
	script := fmt.Sprintf(`display dialog %s with title %s`, quoteForAppleScript(message), quoteForAppleScript(title))
	cmd := exec.Command("osascript", "-e", script)
	cmd.Run()
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

func runAppleScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
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
