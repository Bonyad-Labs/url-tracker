import Foundation
import AppKit

// Simple test runner for AppViewModel
// We compile this with Models.swift and AppViewModel.swift

@main
struct AppViewModelTests {
    static func main() {
        _ = NSApplication.shared // Initialize NSApp for the view model
        func assert(_ condition: Bool, _ message: String) {
            if !condition {
                print("❌ ASSERTION FAILED: \(message)")
                exit(1)
            }
        }

        print("🚀 Starting AppViewModel Tests...")

        // 1. Test Initialization
        let vm = AppViewModel(items: [], entries: [], mode: .dashboard)
        assert(vm.mode == .dashboard, "Initial mode should be dashboard")
        assert(vm.searchEntries.isEmpty, "Initial entries should be empty")

        // 2. Test IPC Command Handling (Search Data)
        let mockEntry = SearchEntry(url: "https://test.com", title: "Test Title", description: "Desc", tags: ["tag1"], category: "Cat", timestamp: 12345)
        let cmd = IPCCommand(mode: .search, searchData: [mockEntry], whitelistData: nil, configData: nil, url: nil, title: nil)

        vm.handleCommand(cmd)

        // Drain the main queue so the async block in handleCommand executes
        RunLoop.main.run(until: Date(timeIntervalSinceNow: 0.1))

        assert(vm.mode == .search, "Mode should update to search")
        assert(vm.searchEntries.count == 1, "Should have 1 search entry")
        assert(vm.searchEntries[0].url == "https://test.com", "Entry URL mismatch")

        // 3. Test Filtering Logic
        vm.searchText = "Title"
        assert(vm.filteredSearchEntries.count == 1, "Should find the entry by title")

        vm.searchText = "Missing"
        assert(vm.filteredSearchEntries.count == 0, "Should not find missing text")

        // 4. Test Category Count
        vm.searchText = ""
        assert(vm.count(for: .category("Cat")) == 1, "Category count mismatch")
        assert(vm.count(for: .category("Other")) == 0, "Empty category should have 0 count")

        // 5. Corner Case: Empty IPC Command
        let emptyCmd = IPCCommand(mode: .dashboard, searchData: nil, whitelistData: nil, configData: nil, url: nil, title: nil)
        vm.handleCommand(emptyCmd)
        RunLoop.main.run(until: Date(timeIntervalSinceNow: 0.1))
        assert(vm.mode == .dashboard, "Mode should revert to dashboard")

        // 6. Corner Case: Search Filtering with Special Characters
        vm.searchEntries = [SearchEntry(url: "https://x.com", title: "Special & Char", description: "", tags: [], category: "", timestamp: 0)]
        vm.searchText = "&"
        assert(vm.filteredSearchEntries.count == 1, "Should find special characters")
        
        vm.sidebarSelection = .untagged
        assert(vm.filteredSearchEntries.count == 1, "Should identify untagged entry")
        
        vm.sidebarSelection = .recentlyAdded
        assert(vm.filteredSearchEntries.count == 0, "Old entry should not be recently added")

        // 7. Test CRUD IPC emission
        // We can't easily capture stdout in this headless test without more complex setup,
        // but we can verify the function calls don't crash.
        vm.deleteEntry(mockEntry)
        vm.updateEntry(mockEntry)

        // 8. Test Inline Editing Flow
        vm.startInlineEdit(mockEntry)
        assert(vm.currentURL == "https://test.com", "URL should match")
        assert(vm.currentTitle == "Test Title", "Title should match")
        assert(vm.saveDescription == "Desc", "Description should match")
        assert(vm.saveCategory == "Cat", "Category should match")
        assert(vm.saveTags == "tag1", "Tags should match")

        vm.saveDescription = "Updated Desc"
        vm.commitInlineEdit(for: mockEntry.url)
        // Note: commitInlineEdit doesn't change mode now, it's handled in the view state
        
        print("✅ All AppViewModel Tests (including Inline Editing) Passed!")
        exit(0)
    }
}
