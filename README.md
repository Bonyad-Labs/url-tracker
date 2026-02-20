# Chrome URL Tracker

A macOS-only background service that monitors Google Chrome tabs and allows you to save and search URLs with custom metadata.

## Features

- **Background Monitoring**: Automatically detects when you navigate to a new URL in Google Chrome.
- **Menu Bar Integration**: Access common actions directly from the macOS menu bar.
- **Unified Premium Manager**: A high-end native SwiftUI interface for all management tasks:
    - **Search**: Interactive, live-filtering search with rich metadata views.
    - **Whitelist Manager**: Manage global exclusions with smart categorization.
    - **Quick Whitelist**: Modern "Domain vs URL" selection dialog for new detections.
- **Native macOS Dialogs**: Promptly asks for metadata using native AppleScript dialogs for lightweight input.
- **Local Storage**: All data is stored locally on your machine in a thread-safe, atomic JSON store.

## Prerequisites

- macOS (Sonoma recommended for best UI)
- Google Chrome
- Go 1.18+ (for building the core)
- Swift (Xcode Command Line Tools) for building native UI components

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
Launch the interactive native search interface:
```bash
./chrome-url-tracker -search
```
This will open a premium SwiftUI window with live-filtering, rich metadata detail view, and native "Open" and "Copy" actions.

## Architecture

- **Go**: Core application logic, concurrency management, and storage.
- **SwiftUI**: Premium native macOS interfaces for complex management tasks.
- **AppleScript (osascript)**: Lightweight macOS UI interactions and Chrome tab monitoring.
- **JSON**: Thread-safe, atomic local storage with automatic backup recovery.

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
