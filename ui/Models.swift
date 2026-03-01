import Foundation

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

enum AppMode: String, Codable {
    case whitelist
    case search
    case add
    case save
    case edit
    case dashboard // New unified mode
    case settings
}

struct ConfigData: Codable {
    let polling_interval: Int
    let storage_path: String
}

struct IPCCommand: Codable {
    let mode: AppMode
    let searchData: [SearchEntry]?
    let whitelistData: [WhitelistItem]?
    let configData: ConfigData?
    let url: String?
    let title: String?
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
