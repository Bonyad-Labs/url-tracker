// Whitelist Manager is a native macOS SwiftUI application used to manage the 
// URL exclusion list and search through saved URLs for the Chrome URL Tracker.
// It provides a modern, searchable UI and exits with the selected value/action to stdout.

import SwiftUI
import AppKit

// MARK: - Data Models

struct WhitelistItem: Identifiable, Codable {
    let id = UUID()
    let value: String
    let type: String      // "domain" or "url"
    let timestamp: Int64  // Unix timestamp
    
    enum CodingKeys: String, CodingKey {
        case value, type, timestamp
    }
}

struct SearchEntry: Identifiable, Codable, Hashable {
    let id = UUID()
    let url: String
    let title: String
    let description: String
    let tags: [String]
    let category: String
    let timestamp: Int64
    
    enum CodingKeys: String, CodingKey {
        case url, title, description, tags, category, timestamp
    }
}

enum AppMode {
    case whitelist
    case search
    case add
    case save
}

enum WhitelistFilter: String, CaseIterable, Identifiable {
    case all = "All"
    case domains = "Domains"
    case urls = "URLs"
    
    var id: String { self.rawValue }
}

enum SidebarSelection: Hashable {
    case all
    case recentlyAdded
    case untagged
    case category(String)
    case tag(String)
}

// MARK: - View Models

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
    
    init(items: [WhitelistItem] = [], entries: [SearchEntry] = [], mode: AppMode = .whitelist, url: String = "", title: String = "") {
        self.whitelistItems = items
        self.searchEntries = entries
        self.mode = mode
        self.currentURL = url
        self.currentTitle = title
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

// MARK: - Components

struct WhitelistRow: View {
    let item: WhitelistItem
    
    var dateString: String {
        let date = Date(timeIntervalSince1970: TimeInterval(item.timestamp))
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        formatter.timeStyle = .none
        return formatter.string(from: date)
    }
    
    var body: some View {
        HStack(spacing: 0) {
            // Type
            HStack {
                Image(systemName: item.type == "url" ? "link" : "globe")
                    .foregroundColor(.blue)
                Text(item.type.capitalized)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .frame(width: 80, alignment: .leading)
            
            // Value
            Text(item.value)
                .font(.system(.body, design: .monospaced))
                .lineLimit(1)
                .truncationMode(.middle)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 8)
            
            // Date
            Text(dateString)
                .font(.caption)
                .foregroundColor(.secondary)
                .frame(width: 80, alignment: .leading)
            
            // Action
            Button(action: {
                print(item.value) // Output to Go via stdout
                NSApplication.shared.terminate(nil)
            }) {
                Image(systemName: "trash")
                    .font(.caption)
            }
            .buttonStyle(.plain)
            .padding(.horizontal, 8)
            .foregroundColor(.red.opacity(0.8))
            .onHover { inside in
                if inside { NSCursor.pointingHand.set() } else { NSCursor.arrow.set() }
            }
        }
        .padding(.vertical, 8)
        .padding(.horizontal, 4)
    }
}

struct SearchRow: View {
    let entry: SearchEntry
    let isSelected: Bool
    
    var faviconURL: URL? {
        // Use Google's favicon service for high-quality icons
        if let domain = URL(string: entry.url)?.host {
            return URL(string: "https://www.google.com/s2/favicons?domain=\(domain)&sz=64")
        }
        return nil
    }
    
    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            // Favicon
            AsyncImage(url: faviconURL) { image in
                image.resizable()
            } placeholder: {
                Image(systemName: "globe")
                    .foregroundColor(.secondary)
            }
            .frame(width: 24, height: 24)
            .cornerRadius(4)
            .padding(.top, 2)
            
            VStack(alignment: .leading, spacing: 4) {
                Text(entry.title.isEmpty ? "No Title" : entry.title)
                    .font(.subheadline)
                    .fontWeight(.bold)
                    .lineLimit(1)
                    .foregroundColor(isSelected ? .white : .primary)
                
                if !entry.description.isEmpty {
                    Text(entry.description)
                        .font(.caption)
                        .foregroundColor(isSelected ? .white.opacity(0.7) : .secondary)
                        .lineLimit(2)
                        .truncationMode(.tail)
                }
                
                HStack {
                    Text(entry.url)
                        .font(.system(size: 10))
                        .foregroundColor(isSelected ? .white.opacity(0.6) : .secondary.opacity(0.8))
                        .lineLimit(1)
                        .truncationMode(.middle)
                    
                    if !entry.category.isEmpty {
                        Spacer()
                        Text(entry.category)
                            .font(.system(size: 9, weight: .bold))
                            .padding(.horizontal, 4)
                            .padding(.vertical, 1)
                            .background(isSelected ? Color.white.opacity(0.2) : Color.blue.opacity(0.1))
                            .foregroundColor(isSelected ? .white : .blue)
                            .cornerRadius(3)
                    }
                }
            }
        }
        .padding(.vertical, 6)
    }
}

struct SidebarRow: View {
    let title: String
    let icon: String
    let selection: SidebarSelection
    @Binding var currentSelection: SidebarSelection
    let count: Int
    
    var isSelected: Bool { selection == currentSelection }
    
    var body: some View {
        Button(action: { currentSelection = selection }) {
            HStack {
                Label(title, systemImage: icon)
                Spacer()
                if count > 0 {
                    Text("\(count)")
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(isSelected ? Color.white.opacity(0.3) : Color.secondary.opacity(0.1))
                        .cornerRadius(10)
                }
            }
            .padding(.vertical, 4)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .foregroundColor(isSelected ? .white : .primary)
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(isSelected ? Color.blue : Color.clear)
        .cornerRadius(6)
    }
}

struct SearchSidebarView: View {
    @ObservedObject var viewModel: AppViewModel
    
    var body: some View {
        List {
            Section("Library") {
                SidebarRow(title: "All URLs", icon: "tray.full", selection: .all, currentSelection: $viewModel.sidebarSelection, count: viewModel.count(for: .all))
                SidebarRow(title: "Recently Added", icon: "clock", selection: .recentlyAdded, currentSelection: $viewModel.sidebarSelection, count: viewModel.count(for: .recentlyAdded))
                SidebarRow(title: "Untagged", icon: "tag.slash", selection: .untagged, currentSelection: $viewModel.sidebarSelection, count: viewModel.count(for: .untagged))
            }
            
            if !viewModel.allCategories.isEmpty {
                Section("Categories") {
                    ForEach(viewModel.allCategories, id: \.self) { cat in
                        SidebarRow(title: cat, icon: "folder", selection: .category(cat), currentSelection: $viewModel.sidebarSelection, count: viewModel.count(for: .category(cat)))
                    }
                }
            }
            
            if !viewModel.allTags.isEmpty {
                Section("Tags") {
                    ForEach(viewModel.allTags, id: \.self) { tag in
                        SidebarRow(title: tag, icon: "tag", selection: .tag(tag), currentSelection: $viewModel.sidebarSelection, count: viewModel.count(for: .tag(tag)))
                    }
                }
            }
        }
        .listStyle(SidebarListStyle())
        .frame(minWidth: 200)
    }
}

// MARK: - Main Views

struct WhitelistView: View {
    @ObservedObject var viewModel: AppViewModel
    
    var body: some View {
        VStack(spacing: 0) {
            header
            
            VStack(spacing: 0) {
                HStack(spacing: 16) {
                    TextField("Search whitelisted items...", text: $viewModel.searchText)
                        .textFieldStyle(.plain)
                        .padding(10)
                        .background(Color.white.opacity(0.05))
                        .cornerRadius(8)
                    
                    Picker("", selection: $viewModel.whitelistFilter) {
                        ForEach(WhitelistFilter.allCases) { filter in
                            Text(filter.rawValue).tag(filter)
                        }
                    }
                    .pickerStyle(.segmented)
                    .frame(width: 180)
                }
                .padding()
                
                // Table Header
                HStack(spacing: 0) {
                    Text("TYPE")
                        .frame(width: 80, alignment: .leading)
                    Text("ENTRY")
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, 8)
                    Text("ADDED ON")
                        .frame(width: 80, alignment: .leading)
                    Text("")
                        .frame(width: 40)
                }
                .font(.system(size: 10, weight: .bold))
                .foregroundColor(.secondary)
                .padding(.horizontal, 24)
                .padding(.vertical, 8)
                .background(Color.black.opacity(0.1))
                
                List {
                    if viewModel.whitelistFilter == .all || viewModel.whitelistFilter == .domains {
                        if !viewModel.filteredDomains.isEmpty {
                            Section(header: Text("Domains").font(.caption).fontWeight(.bold).foregroundColor(.blue)) {
                                ForEach(viewModel.filteredDomains) { item in
                                    WhitelistRow(item: item)
                                        .listRowBackground(Color.clear)
                                }
                            }
                        }
                    }
                    
                    if viewModel.whitelistFilter == .all || viewModel.whitelistFilter == .urls {
                        if !viewModel.filteredWhitelistedURLs.isEmpty {
                            Section(header: Text("Specific URLs").font(.caption).fontWeight(.bold).foregroundColor(.blue)) {
                                ForEach(viewModel.filteredWhitelistedURLs) { item in
                                    WhitelistRow(item: item)
                                        .listRowBackground(Color.clear)
                                }
                            }
                        }
                    }
                    
                    if viewModel.whitelistItems.isEmpty {
                        Text("No whitelisted items yet.")
                            .foregroundColor(.secondary)
                            .frame(maxWidth: .infinity, alignment: .center)
                            .padding(.top, 40)
                    }
                }
                .listStyle(InsetListStyle())
            }
        }
        .background(Color.black.opacity(0.02))
    }
    
    var header: some View {
        HStack {
            Text("Whitelist Manager")
                .font(.headline)
                .foregroundColor(.secondary)
            Spacer()
            Button("Done") {
                NSApplication.shared.terminate(nil)
            }
            .keyboardShortcut(.defaultAction)
        }
        .padding()
        .background(Color(NSColor.windowBackgroundColor).opacity(0.5))
    }
}

struct SearchView: View {
    @ObservedObject var viewModel: AppViewModel
    
    var groupedEntries: [(String, [SearchEntry])] {
        let entries = viewModel.filteredSearchEntries
        if entries.isEmpty { return [] }
        
        var groups: [(String, [SearchEntry])] = []
        let calendar = Calendar.current
        
        let todayEntries = entries.filter { calendar.isDateInToday(Date(timeIntervalSince1970: TimeInterval($0.timestamp))) }
        if !todayEntries.isEmpty { groups.append(("Today", todayEntries)) }
        
        let yesterdayEntries = entries.filter { calendar.isDateInYesterday(Date(timeIntervalSince1970: TimeInterval($0.timestamp))) }
        if !yesterdayEntries.isEmpty { groups.append(("Yesterday", yesterdayEntries)) }
        
        let earlierEntries = entries.filter { 
            let date = Date(timeIntervalSince1970: TimeInterval($0.timestamp))
            return !calendar.isDateInToday(date) && !calendar.isDateInYesterday(date)
        }
        if !earlierEntries.isEmpty { groups.append(("Earlier", earlierEntries)) }
        
        return groups
    }
    
    var body: some View {
        NavigationSplitView {
            SearchSidebarView(viewModel: viewModel)
                .navigationTitle("Library")
        } content: {
            VStack(spacing: 0) {
                // Search Bar
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("Search...", text: $viewModel.searchText)
                        .textFieldStyle(.plain)
                }
                .padding(10)
                .background(Color.secondary.opacity(0.1))
                .cornerRadius(10)
                .padding()
                
                List(selection: $viewModel.selectedEntry) {
                    ForEach(groupedEntries, id: \.0) { group in
                        Section(header: Text(group.0).font(.caption).fontWeight(.bold)) {
                            ForEach(group.1) { entry in
                                SearchRow(entry: entry, isSelected: viewModel.selectedEntry?.id == entry.id)
                                    .tag(entry)
                            }
                        }
                    }
                    
                    if viewModel.filteredSearchEntries.isEmpty {
                        Text("No results found")
                            .foregroundColor(.secondary)
                            .frame(maxWidth: .infinity, alignment: .center)
                            .padding(.top, 40)
                    }
                }
                .listStyle(InsetListStyle())
            }
            .frame(minWidth: 300)
            .navigationTitle("Results")
        } detail: {
            if let entry = viewModel.selectedEntry {
                SearchDetailView(entry: entry)
            } else {
                VStack(spacing: 12) {
                    Image(systemName: "doc.text.magnifyingglass")
                        .font(.system(size: 48))
                        .foregroundColor(.secondary.opacity(0.5))
                    Text("Select an item to view details")
                        .font(.headline)
                        .foregroundColor(.secondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(Color(NSColor.windowBackgroundColor))
            }
        }
    }
}

struct SearchDetailView: View {
    let entry: SearchEntry
    
    var relativeDate: String {
        let date = Date(timeIntervalSince1970: TimeInterval(entry.timestamp))
        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .full
        return formatter.localizedString(for: date, relativeTo: Date())
    }
    
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                VStack(alignment: .leading, spacing: 8) {
                    Text(entry.title.isEmpty ? "No Title" : entry.title)
                        .font(.system(size: 28, weight: .bold, design: .rounded))
                    
                    Button(action: {
                        print("OPEN|\(entry.url)")
                        NSApplication.shared.terminate(nil)
                    }) {
                        Text(entry.url)
                            .font(.body)
                            .foregroundColor(.blue)
                            .underline()
                    }
                    .buttonStyle(.plain)
                }
                
                HStack(spacing: 16) {
                    if !entry.category.isEmpty {
                        Label(entry.category, systemImage: "folder.fill")
                            .font(.caption)
                            .fontWeight(.bold)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(Color.blue.opacity(0.1))
                            .foregroundColor(.blue)
                            .cornerRadius(6)
                    }
                    
                    Text("Added \(relativeDate)")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                
                if !entry.tags.isEmpty {
                    FlowLayout(spacing: 8) {
                        ForEach(entry.tags, id: \.self) { tag in
                            Label(tag, systemImage: "tag")
                                .font(.caption)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(Color.secondary.opacity(0.1))
                                .cornerRadius(6)
                        }
                    }
                }
                
                Divider()
                
                VStack(alignment: .leading, spacing: 12) {
                    Text("DESCRIPTION")
                        .font(.system(size: 11, weight: .black))
                        .foregroundColor(.secondary)
                    
                    Text(entry.description.isEmpty ? "No description provided." : entry.description)
                        .font(.body)
                        .lineSpacing(4)
                }
                
                Spacer(minLength: 40)
                
                HStack(spacing: 12) {
                    Button(action: {
                        print("OPEN|\(entry.url)")
                        NSApplication.shared.terminate(nil)
                    }) {
                        Label("Open in Chrome", systemImage: "safari.fill")
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.large)
                    
                    Button(action: {
                        print("COPY|\(entry.url)")
                        NSApplication.shared.terminate(nil)
                    }) {
                        Label("Copy URL", systemImage: "doc.on.doc.fill")
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.large)
                }
            }
            .padding(32)
        }
        .background(Color(NSColor.windowBackgroundColor))
    }
}

// Simple FlowLayout for Tags
struct FlowLayout: View {
    let spacing: CGFloat
    let children: [AnyView]
    
    init<Views: View>(spacing: CGFloat = 8, @ViewBuilder content: () -> Views) {
        self.spacing = spacing
        // This is a simplified version for demonstration
        self.children = [AnyView(content())]
    }
    
    var body: some View {
        HStack(spacing: spacing) {
            ForEach(0..<children.count, id: \.self) { i in
                children[i]
            }
        }
    }
}

struct AddView: View {
    @ObservedObject var viewModel: AppViewModel
    
    var domain: String {
        URL(string: viewModel.currentURL)?.host ?? viewModel.currentURL
    }
    
    var body: some View {
        VStack(spacing: 24) {
            VStack(spacing: 8) {
                Image(systemName: "shield.checkered")
                    .font(.system(size: 48))
                    .foregroundColor(.blue)
                    .padding(.bottom, 8)
                
                Text("Add to Whitelist")
                    .font(.title2)
                    .fontWeight(.bold)
                
                Text(viewModel.currentTitle.isEmpty ? "New URL Detected" : viewModel.currentTitle)
                    .font(.headline)
                    .lineLimit(1)
                    .foregroundColor(.secondary)
            }
            .padding(.top, 8)
            
            VStack(spacing: 12) {
                // Domain Option
                Button(action: {
                    print(domain)
                    NSApplication.shared.terminate(nil)
                }) {
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Whitelist Domain")
                                .font(.headline)
                            Text(domain)
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                        Spacer()
                        Image(systemName: "globe")
                            .font(.title2)
                    }
                    .padding()
                    .frame(maxWidth: .infinity)
                    .background(Color.blue.opacity(0.1))
                    .cornerRadius(12)
                }
                .buttonStyle(.plain)
                
                // URL Option
                Button(action: {
                    print(viewModel.currentURL)
                    NSApplication.shared.terminate(nil)
                }) {
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Whitelist Specific URL")
                                .font(.headline)
                            Text(viewModel.currentURL)
                                .font(.caption)
                                .lineLimit(1)
                                .truncationMode(.middle)
                                .foregroundColor(.secondary)
                        }
                        Spacer()
                        Image(systemName: "link")
                            .font(.title2)
                    }
                    .padding()
                    .frame(maxWidth: .infinity)
                    .background(Color.secondary.opacity(0.1))
                    .cornerRadius(12)
                }
                .buttonStyle(.plain)
            }
            
            HStack {
                Button("Cancel") {
                    NSApplication.shared.terminate(nil)
                }
                .buttonStyle(.plain)
                .foregroundColor(.secondary)
                
                Spacer()
            }
        }
        .padding(32)
        .frame(width: 400)
    }
}

struct SaveView: View {
    @ObservedObject var viewModel: AppViewModel
    @FocusState private var focusedField: Field?
    
    enum Field {
        case description, category, tags
    }
    
    var body: some View {
        VStack(spacing: 24) {
            // Header
            VStack(spacing: 8) {
                Image(systemName: "square.and.pencil")
                    .font(.system(size: 40))
                    .foregroundColor(.blue)
                
                Text("Save New URL")
                    .font(.title2)
                    .fontWeight(.bold)
                
                Text(viewModel.currentTitle.isEmpty ? viewModel.currentURL : viewModel.currentTitle)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .lineLimit(1)
            }
            .padding(.top, 8)
            
            // Form
            VStack(alignment: .leading, spacing: 16) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("Description")
                        .font(.caption)
                        .fontWeight(.semibold)
                        .foregroundColor(.secondary)
                    TextField("What is this page about?", text: $viewModel.saveDescription)
                        .textFieldStyle(.plain)
                        .padding(10)
                        .background(Color.white.opacity(0.05))
                        .cornerRadius(8)
                        .focused($focusedField, equals: .description)
                }
                
                HStack(spacing: 16) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Category")
                            .font(.caption)
                            .fontWeight(.semibold)
                            .foregroundColor(.secondary)
                        TextField("Research, Social...", text: $viewModel.saveCategory)
                            .textFieldStyle(.plain)
                            .padding(10)
                            .background(Color.white.opacity(0.05))
                            .cornerRadius(8)
                            .focused($focusedField, equals: .category)
                    }
                    
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Tags")
                            .font(.caption)
                            .fontWeight(.semibold)
                            .foregroundColor(.secondary)
                        TextField("tag1, tag2...", text: $viewModel.saveTags)
                            .textFieldStyle(.plain)
                            .padding(10)
                            .background(Color.white.opacity(0.05))
                            .cornerRadius(8)
                            .focused($focusedField, equals: .tags)
                    }
                }
            }
            
            // Actions
            VStack(spacing: 12) {
                Button(action: {
                    let response = [
                        "action": "save",
                        "description": viewModel.saveDescription,
                        "category": viewModel.saveCategory,
                        "tags": viewModel.saveTags
                    ]
                    if let jsonData = try? JSONEncoder().encode(response),
                       let jsonString = String(data: jsonData, encoding: .utf8) {
                        print(jsonString)
                        NSApplication.shared.terminate(nil)
                    }
                }) {
                    Text("Save Entry")
                        .fontWeight(.bold)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .background(Color.blue)
                        .foregroundColor(.white)
                        .cornerRadius(10)
                }
                .buttonStyle(.plain)
                .keyboardShortcut(.return, modifiers: [])
                
                HStack(spacing: 12) {
                    Button(action: {
                        print("{\"action\": \"whitelist\"}")
                        NSApplication.shared.terminate(nil)
                    }) {
                        Text("Whitelist...")
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(Color.white.opacity(0.1))
                            .cornerRadius(8)
                    }
                    .buttonStyle(.plain)
                    
                    Button(action: {
                        NSApplication.shared.terminate(nil)
                    }) {
                        Text("Skip")
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(Color.white.opacity(0.05))
                            .cornerRadius(8)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .padding(32)
        .frame(width: 450)
        .onAppear {
            focusedField = .description
        }
    }
}

struct MainContentView: View {
    @StateObject var viewModel: AppViewModel
    
    var body: some View {
        Group {
            switch viewModel.mode {
            case .whitelist:
                WhitelistView(viewModel: viewModel)
            case .search:
                SearchView(viewModel: viewModel)
            case .add:
                AddView(viewModel: viewModel)
            case .save:
                SaveView(viewModel: viewModel)
            }
        }
        .frame(minWidth: (viewModel.mode == .add || viewModel.mode == .save) ? 400 : 600, 
               minHeight: (viewModel.mode == .add || viewModel.mode == .save) ? 350 : 450)
    }
}

// MARK: - App Delegate

class AppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate {
    var window: NSWindow!
    var viewModel: AppViewModel!

    func applicationDidFinishLaunching(_ notification: Notification) {
        let contentView = MainContentView(viewModel: self.viewModel)

        window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, 
                               width: (viewModel.mode == .add || viewModel.mode == .save) ? 450 : (viewModel.mode == .search ? 900 : 700), 
                               height: (viewModel.mode == .add || viewModel.mode == .save) ? 450 : 600),
            styleMask: (viewModel.mode == .add || viewModel.mode == .save) ? [.titled, .closable, .fullSizeContentView] : [.titled, .closable, .miniaturizable, .resizable, .fullSizeContentView],
            backing: .buffered, defer: false)
        
        if viewModel.mode != .add && viewModel.mode != .save {
            window.minSize = NSSize(width: viewModel.mode == .search ? 850 : 600, height: 450)
        }
        
        window.center()
        window.titleVisibility = viewModel.mode == .search ? .hidden : .visible
        window.titlebarAppearsTransparent = viewModel.mode == .search
        window.contentView = NSHostingView(rootView: contentView)
        window.delegate = self
        window.makeKeyAndOrderFront(nil)
        
        NSApp.activate(ignoringOtherApps: true)
    }
    
    func windowWillClose(_ notification: Notification) {
        NSApplication.shared.terminate(nil)
    }
    
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return true
    }
}

// MARK: - Main

let args = CommandLine.arguments
var mode: AppMode = .whitelist
var whitelistItems: [WhitelistItem] = []
var entries: [SearchEntry] = []
var urlParam: String = ""
var titleParam: String = ""

// Primitive argument parsing
for (index, arg) in args.enumerated() {
    if arg == "--mode" && index + 1 < args.count {
        switch args[index+1] {
        case "search": mode = .search
        case "add": mode = .add
        case "save": mode = .save
        default: mode = .whitelist
        }
    }
    if arg == "--url" && index + 1 < args.count {
        urlParam = args[index+1]
    }
    if arg == "--title" && index + 1 < args.count {
        titleParam = args[index+1]
    }
    if arg == "--data" && index + 1 < args.count {
        let dataStr = args[index+1]
        guard let data = dataStr.data(using: .utf8) else { continue }
        
        if mode == .whitelist {
            if let decoded = try? JSONDecoder().decode([WhitelistItem].self, from: data) {
                whitelistItems = decoded
            }
        } else if mode == .search {
            if let decoded = try? JSONDecoder().decode([SearchEntry].self, from: data) {
                entries = decoded
            }
        }
    }
}

let app = NSApplication.shared
let delegate = AppDelegate()
delegate.viewModel = AppViewModel(items: whitelistItems, entries: entries, mode: mode, url: urlParam, title: titleParam)

app.delegate = delegate
app.setActivationPolicy(.regular)
app.run()
