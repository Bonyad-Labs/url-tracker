import SwiftUI
import AppKit

// MARK: - Data Model
struct WhitelistItem: Identifiable {
    let id = UUID()
    let value: String
}

// MARK: - View Model
class WhitelistViewModel: ObservableObject {
    @Published var items: [WhitelistItem]
    @Published var searchText = ""
    
    init(items: [String]) {
        self.items = items.map { WhitelistItem(value: $0) }
    }
    
    private func isURL(_ value: String) -> Bool {
        return value.contains("://") || (value.contains("/") && !value.hasSuffix("/"))
    }
    
    var filteredDomains: [WhitelistItem] {
        items.filter { !isURL($0.value) && (searchText.isEmpty || $0.value.lowercased().contains(searchText.lowercased())) }
    }
    
    var filteredURLs: [WhitelistItem] {
        items.filter { isURL($0.value) && (searchText.isEmpty || $0.value.lowercased().contains(searchText.lowercased())) }
    }
}

// MARK: - Main View
struct WhitelistManagerView: View {
    @StateObject var viewModel: WhitelistViewModel
    
    var body: some View {
        VStack(spacing: 0) {
            // Custom Header
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
            
            // Search Bar
            TextField("Search...", text: $viewModel.searchText)
                .textFieldStyle(RoundedBorderTextFieldStyle())
                .padding(.horizontal)
                .padding(.bottom, 8)
            
            // List with Sections
            List {
                if !viewModel.filteredDomains.isEmpty {
                    Section(header: Text("Domains").font(.caption).fontWeight(.bold)) {
                        ForEach(viewModel.filteredDomains) { item in
                            WhitelistRow(item: item)
                        }
                    }
                }
                
                if !viewModel.filteredURLs.isEmpty {
                    Section(header: Text("Specific URLs").font(.caption).fontWeight(.bold)) {
                        ForEach(viewModel.filteredURLs) { item in
                            WhitelistRow(item: item)
                        }
                    }
                }
            }
            .listStyle(InsetListStyle())
        }
        .frame(width: 450, height: 400)
    }
}

struct WhitelistRow: View {
    let item: WhitelistItem
    
    var body: some View {
        HStack {
            Image(systemName: item.value.contains("://") ? "link" : "globe")
                .foregroundColor(.blue)
                .frame(width: 20)
            Text(item.value)
                .font(.system(.body, design: .monospaced))
                .lineLimit(1)
                .truncationMode(.middle)
            Spacer()
            Button(action: {
                print(item.value) // Output to Go via stdout
                NSApplication.shared.terminate(nil)
            }) {
                Text("Remove")
                    .font(.caption)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2)
            }
            .buttonStyle(.bordered)
            .tint(.red)
        }
        .padding(.vertical, 2)
    }
}

// MARK: - App Delegate
class AppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate {
    var window: NSWindow!
    var items: [String] = []

    func applicationDidFinishLaunching(_ notification: Notification) {
        let contentView = WhitelistManagerView(viewModel: WhitelistViewModel(items: self.items))

        window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 450, height: 400),
            styleMask: [.titled, .closable, .miniaturizable, .fullSizeContentView],
            backing: .buffered, defer: false)
        window.center()
        window.title = "Manage Whitelist"
        window.contentView = NSHostingView(rootView: contentView)
        window.delegate = self
        window.makeKeyAndOrderFront(nil)
        
        // Ensure app stays in front
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
let app = NSApplication.shared
let delegate = AppDelegate()

// Parse arguments (JSON array passed from Go)
let args = CommandLine.arguments
if args.count > 1 {
    let data = args[1].data(using: .utf8)!
    if let decoded = try? JSONDecoder().decode([String].self, from: data) {
        delegate.items = decoded
    }
}

app.delegate = delegate
app.setActivationPolicy(.regular)
app.run()
