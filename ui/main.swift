import AppKit

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
        case "settings": mode = .settings
        case "dashboard": mode = .dashboard
        default: mode = .dashboard
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
