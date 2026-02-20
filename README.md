# Chrome URL Tracker

A macOS-only background service that monitors Google Chrome tabs and allows you to save and search URLs with custom metadata.

## Features

- **Background Monitoring**: Automatically detects when you navigate to a new URL in Google Chrome.
- **Menu Bar Integration**: Access common actions directly from the macOS menu bar.
- **Native macOS Dialogs**: Promptly asks for a description, category, and tags using native macOS dialogs.
- **Interactive Search**: Search through your saved URLs using a CLI interface or menu bar shortcut.
- **Dynamic Whitelisting**: Exclude domains in real-time via the menu bar.
- **Local Storage**: All data is stored locally on your machine in a JSON file.

## Prerequisites

- macOS
- Google Chrome
- Go 1.18+ (for building)

## Getting Started

### Installation

1. Clone the repository to your local machine.
2. Run the update script to build and install the service:
   ```bash
   ./update.sh
   ```
3. **Grant Permissions**: 
   - When prompted or via System Settings, grant your Terminal/Binary permission for **Automation** (to control Google Chrome) and **Accessibility** (to display dialogs).

### Usage

#### Monitor Mode (Default)
The service runs in the background. You can also start it manually for testing:
```bash
./chrome-url-tracker
```
Optional flags:
- `-interval 1000`: Set polling interval in milliseconds (default 1000ms).
- `-storage ~/custom-path.json`: Specify a custom storage location.

#### Search Mode
Launch the interactive search interface:
```bash
./chrome-url-tracker -search
```
This will prompt you for a search query and display a list of results to open or copy.

## Architecture

- **Go**: Core application logic.
- **AppleScript (osascript)**: Used for all macOS UI interactions and Chrome tab monitoring.
- **JSON**: Simple, human-readable local storage.

## Project Structure

- `main.go`: Application coordination and mode selection.
- `monitor/`: Chrome tab polling logic.
- `storage/`: JSON data persistence.
- `ui/`: macOS dialog orchestration.
- `update.sh`: Build and LaunchAgent management script.

## Future Improvements

- **Premium Web Dashboard**: A high-end browser-based interface for managing and visualizing saved URLs with filters and analytics.
- **Global Hotkeys**: Custom keyboard shortcuts to trigger search or quick-save.
- **Safari Support**: Extend monitoring to other browsers.

## License

MIT
