import SwiftUI
import UniformTypeIdentifiers

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

struct WhitelistView: View {
    @ObservedObject var viewModel: AppViewModel
    @State private var newWhitelistURL: String = ""
    
    var body: some View {
        VStack(spacing: 0) {
            header
            
            VStack(spacing: 0) {
                HStack(spacing: 12) {
                    TextField("Add domain or URL to whitelist...", text: $newWhitelistURL)
                        .textFieldStyle(.plain)
                        .padding(10)
                        .background(Color.white.opacity(0.05))
                        .cornerRadius(8)

                    Button(action: {
                        if !newWhitelistURL.isEmpty {
                            print("ADD_WHITELIST|\(newWhitelistURL)")
                            fflush(stdout)
                            newWhitelistURL = ""
                        }
                    }) {
                        Text("Add")
                            .fontWeight(.bold)       
                            .padding(.horizontal, 20)
                            .padding(.vertical, 10)
                            .background(Color.blue)
                            .foregroundColor(.white)
                            .cornerRadius(8)
                    }
                    .buttonStyle(.plain)
                    .keyboardShortcut(.return, modifiers: [])
                }
                .padding(.horizontal)
                .padding(.top, 16)
                .padding(.bottom, 8)
                
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
                        fflush(stdout)
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
                        fflush(stdout)
                    }) {
                        Label("Open in Chrome", systemImage: "safari.fill")
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.large)
                    
                    Button(action: {
                        print("COPY|\(entry.url)")
                        fflush(stdout)
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

struct AddView: View {
    @ObservedObject var viewModel: AppViewModel
    
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

            VStack(alignment: .leading, spacing: 16) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("URL or Domain to Whitelist")
                        .font(.caption)
                        .fontWeight(.semibold)
                        .foregroundColor(.secondary)
                    TextField("Enter domain or URL...", text: $viewModel.currentURL)
                        .textFieldStyle(.plain)
                        .padding(10)
                        .background(Color.white.opacity(0.05))
                        .cornerRadius(8)
                }
                
                VStack(alignment: .leading, spacing: 6) {
                    Text("Title (Optional)")
                        .font(.caption)
                        .fontWeight(.semibold)
                        .foregroundColor(.secondary)
                    TextField("Enter title...", text: $viewModel.currentTitle)
                        .textFieldStyle(.plain)
                        .padding(10)
                        .background(Color.white.opacity(0.05))
                        .cornerRadius(8)
                }
            }
            
            HStack(spacing: 12) {
                Button("Cancel") {
                    print("CANCEL|")
                    fflush(stdout)
                    viewModel.mode = .dashboard
                    if NSApp.windows.first?.styleMask.contains(.titled) == true {
                        NSApp.windows.first?.close()
                    }
                }
                .buttonStyle(.plain)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 12)
                .background(Color.secondary.opacity(0.1))
                .cornerRadius(10)
                
                Button(action: {
                    print("ADD_WHITELIST|\(viewModel.currentURL)")
                    fflush(stdout)
                    viewModel.mode = .dashboard
                    if NSApp.windows.first?.styleMask.contains(.titled) == true {
                        NSApp.windows.first?.close()
                    }
                }) {
                    Text("Save")
                        .fontWeight(.bold)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .background(Color.blue)
                        .foregroundColor(.white)
                        .cornerRadius(10)
                }
                .buttonStyle(.plain)
                .keyboardShortcut(.return, modifiers: [])
            }
        }
        .padding(32)
        .frame(width: 450)
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
                        print("SAVE_ENTRY|\(jsonString)")
                        fflush(stdout)
                        if NSApp.windows.first?.styleMask.contains(.titled) == true {
                            NSApp.windows.first?.close()
                        }
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
                        print("ACTION_WHITELIST|")
                        fflush(stdout)
                        if NSApp.windows.first?.styleMask.contains(.titled) == true {
                            NSApp.windows.first?.close()
                        }
                    }) {
                        Text("Whitelist...")
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(Color.white.opacity(0.1))
                            .cornerRadius(8)
                    }
                    .buttonStyle(.plain)
                    
                    Button(action: {
                        print("ACTION_SKIP|")
                        fflush(stdout)
                        if NSApp.windows.first?.styleMask.contains(.titled) == true {
                            NSApp.windows.first?.close()
                        }
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

struct UnifiedDashboardView: View {
    @ObservedObject var viewModel: AppViewModel
    
    var body: some View {
        TabView(selection: Binding(
            get: { self.viewModel.mode == .whitelist ? 1 : 0 },
            set: { self.viewModel.mode = $0 == 1 ? .whitelist : .search }
        )) {
            SearchView(viewModel: viewModel)
                .tabItem {
                    Label("Saved URLs", systemImage: "magnifyingglass")
                }
                .tag(0)
            
            WhitelistView(viewModel: viewModel)
                .tabItem {
                    Label("Whitelist", systemImage: "shield.checkered")
                }
                .tag(1)
        }
        .padding()
    }
}

struct SettingsView: View {
    @ObservedObject var viewModel: AppViewModel
    
    var body: some View {
        VStack(alignment: .leading, spacing: 24) {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Image(systemName: "gearshape.fill")
                        .font(.system(size: 32))
                        .foregroundColor(.blue)
                    Text("Settings")
                        .font(.largeTitle)
                        .fontWeight(.bold)
                }
                Text("Manage configuration and data")
                    .foregroundColor(.secondary)
            }
            .padding(.bottom, 8)
            
            GroupBox("Configuration") {
                VStack(alignment: .leading, spacing: 16) {
                    HStack {
                        Text("Polling Interval (ms):")
                            .frame(width: 140, alignment: .leading)
                        TextField("1000", text: $viewModel.pollingInterval)
                            .textFieldStyle(.roundedBorder)
                            .frame(width: 100)
                        Spacer()
                    }
                    
                    HStack {
                        Text("Storage Path:")
                            .frame(width: 140, alignment: .leading)
                        Text(viewModel.storagePath)
                            .font(.system(.caption, design: .monospaced))
                            .foregroundColor(.secondary)
                            .lineLimit(1)
                            .truncationMode(.middle)
                    }
                    
                    Button("Save Configuration") {
                        if let interval = Int(viewModel.pollingInterval) {
                            print("SAVE_CONFIG|\(interval)")
                            fflush(stdout)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                }
                .padding()
            }
            
            GroupBox("Data Management") {
                VStack(alignment: .leading, spacing: 16) {
                    Text("Import or export your saved URLs to standard Netscape Bookmark HTML format.")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    
                    HStack(spacing: 16) {
                        Menu {
                            Button("From Browser Bookmarks (.html)") {
                                importData(type: "HTML")
                            }
                            Button("From Native Backup (.json)") {
                                importData(type: "JSON")
                            }
                        } label: {
                            Label("Import...", systemImage: "square.and.arrow.down")
                        }
                        .menuStyle(.borderlessButton)
                        .fixedSize()
                        
                        Menu {
                            Button("To Browser Bookmarks (.html)") {
                                exportData(type: "HTML")
                            }
                            Button("To Native Backup (.json)") {
                                exportData(type: "JSON")
                            }
                        } label: {
                            Label("Export...", systemImage: "square.and.arrow.up")
                        }
                        .menuStyle(.borderlessButton)
                        .fixedSize()
                    }
                    
                    Text("Note: Because standard browser bookmarks do not support Tags, Chrome URL Tracker uses Categories as Bookmark Folders for cross-compatibility. Tags are skipped on export.")
                        .font(.caption2)
                        .foregroundColor(.secondary.opacity(0.8))
                        .padding(.top, 4)
                }
                .padding()
            }
            
            Spacer()
        }
        .padding(32)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }
    
    func importData(type: String) {
        let panel = NSOpenPanel()
        panel.allowsMultipleSelection = false
        panel.canChooseDirectories = false
        panel.canChooseFiles = true
        
        if type == "HTML" {
            panel.allowedContentTypes = [.html]
        } else {
            panel.allowedContentTypes = [.json]
        }
        
        if panel.runModal() == .OK, let url = panel.url {
            if type == "HTML" {
                print("IMPORT_BOOKMARKS|\(url.path)")
            } else {
                print("IMPORT_JSON|\(url.path)")
            }
            fflush(stdout)
        }
    }
    
    func exportData(type: String) {
        let panel = NSSavePanel()
        panel.canCreateDirectories = true
        
        if type == "HTML" {
            panel.nameFieldStringValue = "chrome-url-tracker-bookmarks.html"
            panel.allowedContentTypes = [.html]
        } else {
            panel.nameFieldStringValue = "chrome-url-tracker-backup.json"
            panel.allowedContentTypes = [.json]
        }
        
        if panel.runModal() == .OK, let url = panel.url {
            if type == "HTML" {
                print("EXPORT_BOOKMARKS|\(url.path)")
            } else {
                print("EXPORT_JSON|\(url.path)")
            }
            fflush(stdout)
        }
    }
}

struct MainContentView: View {
    @ObservedObject var viewModel: AppViewModel
    
    var body: some View {
        Group {
            switch viewModel.mode {
            case .whitelist, .search, .dashboard:
                UnifiedDashboardView(viewModel: viewModel)
            case .add:
                AddView(viewModel: viewModel)
            case .save:
                SaveView(viewModel: viewModel)
            case .settings:
                SettingsView(viewModel: viewModel)
            }
        }
        .frame(minWidth: (viewModel.mode == .add || viewModel.mode == .save || viewModel.mode == .settings) ? 400 : 800, 
               minHeight: (viewModel.mode == .add || viewModel.mode == .save || viewModel.mode == .settings) ? 350 : 500)
    }
}
