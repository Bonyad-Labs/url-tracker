# Chrome URL Tracker

A macOS-only background service that monitors Google Chrome and Safari tabs, allowing you to save and search URLs with custom metadata.

## Features

- **Background Monitoring**: Automatically detects when you navigate to a new URL in Google Chrome or Safari. URL query parameters and fragments are ignored to prevent duplicates.
- **Menu Bar Integration**: Access common actions directly from the macOS menu bar.
- **Unified Premium Manager**: A high-end native SwiftUI interface for all management tasks:
    - **Organized Search**: High-end **3-column manager** (Sidebar → List → Detail) with:
        - **Sidebar**: Smart filtering by Categories, Tags, and Library folders (Recently Added, Untagged).
        - **Temporal Headers**: Automatic chronological grouping (Today, Yesterday, Earlier).
        - **Visual Scanning**: High-quality site favicons and row previews (snippet-style).
    - **Save Entry**: Premium single-window form for capturing descriptions, categories, and tags.
    - **Whitelist Manager**: Professional tabular view with type icons, persistent metadata (Date Added), and segmented toggles (All/Domains/URLs).
    - **Quick Whitelist**: Modern "Domain vs URL" selection dialog for new detections.
    - **Pause Monitoring**: Native toggle to temporarily suspend URL tracking.
- **Native Notifications**: Real-time macOS system notifications for all background actions.
- **Local Storage**: All data is stored locally in a high-performance **SQLite** database in the standard macOS `Application Support` directory.

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
   - When prompted or via System Settings, grant your Terminal/Binary permission for **Automation** (to control Google Chrome and Safari) and **Accessibility** (to display dialogs).

### Usage

#### Monitor Mode (Default)
The service runs in the background, monitoring your browser tabs. When a new URL is detected, it presents a **unified native SwiftUI form** to quickly save metadata or whitelist the domain.

To run manually for testing:
```bash
~/Applications/ChromeURLTracker.app/Contents/MacOS/chrome-url-tracker
```
(Note: Polling interval and storage paths are now fully managed via the in-app "Preferences..." window, which saves to `~/Library/Application Support/chrome-url-tracker/config.json`).

Launch the interactive native search interface directly:
```bash
~/Applications/ChromeURLTracker.app/Contents/MacOS/chrome-url-tracker --mode search
```
This will open a professional **3-column organized manager** with:
- **Navigation Sidebar**: Quick access to smart folders, categories, and tags.
- **Grouped List**: Results organized by date (Today/Yesterday/Earlier) with site icons.
- **Rich Detail View**: Complete metadata display with interactive Open/Copy actions.

### Data Management (Import & Export)

You can freely import and export URLs from the "Preferences..." menu. Chrome URL Tracker uses two formats for importing and exporting URLs. The standard **Netscape Bookmark HTML format**, meaning it is 100% compatible with Chrome, Safari, and Firefox. And a **JSON format** that is not supported by other browsers.

**  HTML Format:**
- **Categories map to Folders.** When you export your data, each `Category` you created becomes a Bookmark Folder (e.g., `<H3>`). When importing from Chrome, links inside a folder are assigned that folder's name as their `Category`.
- **Tags are ignored.** Because standard browser bookmarks do not natively support tags, any tags you have in Chrome URL Tracker are dropped upon export to ensure the HTML file remains strictly compliant. Imported bookmarks will have an empty tags array.

**JSON Format:**
- **Categories map to Categories.** When you export your data, each `Category` you created becomes a Category (e.g., `"category": "Research"`). When importing from Chrome, links inside a folder are assigned that folder's name as their `Category`.
- **Tags are preserved.** Because standard browser bookmarks do not natively support tags, any tags you have in Chrome URL Tracker are dropped upon export to ensure the HTML file remains strictly compliant. Imported bookmarks will have an empty tags array.

### Testing

The project includes an automated unit test suite covering storage, monitor parsing, and UI utilities.

Run all tests:
```bash
go test -v ./...
```

Check coverage (Storage package):
```bash
go test -cover ./storage
```

## Architecture

- **Go**: Core application logic, concurrency management, and storage orchestration.
- **SQLite**: Local relational storage via pure Go driver (`modernc.org/sqlite`) for high-performance searching and filtering.
- **SwiftUI**: Premium native macOS interfaces for all primary user interactions (Search, Save, Whitelist).
- **AppleScript (osascript)**: Lightweight hooks for Chrome tab monitoring and system notifications.

## Project Structure

- `main.go`: Application coordination, mode selection, and database lifecycle management.
- `monitor/`: Browser tab polling and active tab detection logic.
- `storage/`: SQLite storage layer with schema management and data persistence.
- `ui/`: macOS dialog orchestration and native SwiftUI bridge.
- `update.sh`: Build and LaunchAgent management script.

## Future Improvements

- **Premium Web Dashboard**: A high-end browser-based interface for managing and visualizing saved URLs with filters and analytics.
- **Global Hotkeys**: Custom keyboard shortcuts to trigger search or quick-save.

## License

MIT
