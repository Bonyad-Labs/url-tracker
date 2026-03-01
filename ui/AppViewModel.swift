import SwiftUI
import AppKit

class AppViewModel: ObservableObject {
    @Published var mode: AppMode = .whitelist
    @Published var searchText = ""
    
    // Whitelist/Search Data
    @Published var whitelistItems: [WhitelistItem] = []
    @Published var whitelistFilter: WhitelistFilter = .all
    @Published var searchEntries: [SearchEntry] = []
    @Published var selectedEntry: SearchEntry?
    @Published var sidebarSelection: SidebarSelection = .all
    
    // Add Mode Data
    @Published var currentURL: String = ""
    @Published var currentTitle: String = ""
    
    // Save Mode Data
    @Published var saveDescription: String = ""
    @Published var saveCategory: String = "Research"
    @Published var saveTags: String = ""
    
    // Settings Data
    @Published var pollingInterval: String = "1000"
    @Published var storagePath: String = ""
    
    init(items: [WhitelistItem] = [], entries: [SearchEntry] = [], mode: AppMode = .dashboard, url: String = "", title: String = "") {
        self.whitelistItems = items
        self.searchEntries = entries
        self.mode = mode
        self.currentURL = url
        self.currentTitle = title
    }
    
    // Process incoming IPC commands from Go
    func handleCommand(_ cmd: IPCCommand) {
        DispatchQueue.main.async {
            self.mode = cmd.mode
            if let entries = cmd.searchData {
                self.searchEntries = entries
            }
            if let items = cmd.whitelistData {
                self.whitelistItems = items
            }
            if let u = cmd.url {
                self.currentURL = u
            }
            if let t = cmd.title {
                self.currentTitle = t
            }
            if let config = cmd.configData {
                self.pollingInterval = String(config.polling_interval)
                self.storagePath = config.storage_path
            }
            
            // Bring app to front
            fputs("DEBUG: Activating window for mode: \(self.mode)\n", stderr)
            
            // NSApp might be nil in headless tests
            if let app = NSApplication.shared as NSApplication? {
                app.setActivationPolicy(.regular) // Show in Dock
                app.activate(ignoringOtherApps: true)
                
                if let window = app.windows.first {
                    if window.isMiniaturized {
                        window.deminiaturize(nil)
                    }
                    window.makeKeyAndOrderFront(nil)
                }
            }
        }
    }
    
    // MARK: - Whitelist Logic
    private func isURL(_ value: String) -> Bool {
        return value.contains("://") || (value.contains("/") && !value.hasSuffix("/"))
    }
    
    var filteredDomains: [WhitelistItem] {
        whitelistItems.filter { $0.type == "domain" && (searchText.isEmpty || $0.value.lowercased().contains(searchText.lowercased())) }
    }
    
    var filteredWhitelistedURLs: [WhitelistItem] {
        whitelistItems.filter { $0.type == "url" && (searchText.isEmpty || $0.value.lowercased().contains(searchText.lowercased())) }
    }
    
    // MARK: - Bookmark Management Logic
    func deleteEntry(_ entry: SearchEntry) {
        print("DELETE_ENTRY|\(entry.url)")
        fflush(stdout)
    }
    
    func updateEntry(_ entry: SearchEntry) {
        if let data = try? JSONEncoder().encode(entry),
           let jsonString = String(data: data, encoding: .utf8) {
            print("UPDATE_ENTRY|\(jsonString)")
            fflush(stdout)
        }
    }
    
    func startInlineEdit(_ entry: SearchEntry) {
        self.saveDescription = entry.description
        self.saveCategory = entry.category
        self.saveTags = entry.tags.joined(separator: ", ")
        self.currentTitle = entry.title
        self.currentURL = entry.url
    }
    
    func commitInlineEdit(for entryId: String) {
        let tags = saveTags.split(separator: ",").map { String($0).trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
        // Find existing timestamp or use current
        let timestamp = searchEntries.first(where: { $0.url == currentURL })?.timestamp ?? Int64(Date().timeIntervalSince1970)
        
        let updatedEntry = SearchEntry(
            url: currentURL,
            title: currentTitle,
            description: saveDescription,
            tags: tags,
            category: saveCategory,
            timestamp: timestamp
        )
        updateEntry(updatedEntry)
    }
    
    // MARK: - Search Sidebar Logic
    var allCategories: [String] {
        Array(Set(searchEntries.compactMap { $0.category.isEmpty ? nil : $0.category })).sorted()
    }
    
    var allTags: [String] {
        Array(Set(searchEntries.flatMap { $0.tags })).sorted()
    }
    
    // MARK: - Search Logic
    var filteredSearchEntries: [SearchEntry] {
        var baseEntries = searchEntries
        
        // 1. Sidebar Filter
        switch sidebarSelection {
        case .all:
            break
        case .recentlyAdded:
            let yesterday = Int64(Date().timeIntervalSince1970) - 86400
            baseEntries = baseEntries.filter { $0.timestamp >= yesterday }
        case .untagged:
            baseEntries = baseEntries.filter { $0.tags.isEmpty }
        case .category(let cat):
            baseEntries = baseEntries.filter { $0.category == cat }
        case .tag(let tag):
            baseEntries = baseEntries.filter { $0.tags.contains(tag) }
        }
        
        // 2. Text Search
        if !searchText.isEmpty {
            let query = searchText.lowercased()
            baseEntries = baseEntries.filter { 
                $0.title.lowercased().contains(query) || 
                $0.url.lowercased().contains(query) || 
                $0.description.lowercased().contains(query) ||
                $0.category.lowercased().contains(query) ||
                $0.tags.contains(where: { $0.lowercased().contains(query) })
            }
        }
        
        return baseEntries.sorted { $0.timestamp > $1.timestamp }
    }
    
    func count(for selection: SidebarSelection) -> Int {
        switch selection {
        case .all: return searchEntries.count
        case .recentlyAdded:
            let yesterday = Int64(Date().timeIntervalSince1970) - 86400
            return searchEntries.filter { $0.timestamp >= yesterday }.count
        case .untagged:
            return searchEntries.filter { $0.tags.isEmpty }.count
        case .category(let cat):
            return searchEntries.filter { $0.category == cat }.count
        case .tag(let tag):
            return searchEntries.filter { $0.tags.contains(tag) }.count
        }
    }
}
