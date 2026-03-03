import SwiftUI
import AppKit
import Carbon

class AppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate {
    var window: NSWindow!
    var viewModel: AppViewModel!

    func applicationDidFinishLaunching(_ notification: Notification) {
        fputs("DEBUG: UI starting in mode: \(self.viewModel.mode)\n", stderr)
        let contentView = MainContentView(viewModel: self.viewModel)

        window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, 
                               width: (viewModel.mode == .add || viewModel.mode == .save) ? 450 : (viewModel.mode == .settings ? 600 : 900), 
                               height: (viewModel.mode == .add || viewModel.mode == .save) ? 450 : (viewModel.mode == .settings ? 550 : 600)),
            styleMask: (viewModel.mode == .add || viewModel.mode == .save || viewModel.mode == .settings) ? [.titled, .closable, .fullSizeContentView] : [.titled, .closable, .miniaturizable, .resizable, .fullSizeContentView],
            backing: .buffered, defer: false)
        
        if viewModel.mode != .add && viewModel.mode != .save && viewModel.mode != .settings {
            window.minSize = NSSize(width: 850, height: 450)
        }
        
        window.center()
        window.titleVisibility = .hidden
        window.titlebarAppearsTransparent = true
        window.contentView = NSHostingView(rootView: contentView)
        window.delegate = self
        
        if viewModel.mode != .dashboard {
            window.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
        }
        
        setupGlobalHotkey()
        startStdinListener()
    }
    
    func setupGlobalHotkey() {
        var hotKey: EventHotKeyRef?
        let hotKeyID = EventHotKeyID(signature: OSType(0x53574654), id: 1) // 'SWFT'
        
        let eventHandler: EventHandlerUPP = { (nextHandler, theEvent, userData) -> OSStatus in
            // When HotKey is pressed, notify Go
            fputs("HOTKEY_SAVE|\n", stdout)
            fflush(stdout)
            return noErr
        }
        
        var eventType = EventTypeSpec(eventClass: OSType(kEventClassKeyboard), eventKind: UInt32(kEventHotKeyPressed))
        
        // Use InstallEventHandler directly as the macro is not available in Swift
        InstallEventHandler(GetApplicationEventTarget(), eventHandler, 1, &eventType, nil, nil)
        
        // Register Cmd+Shift+S (S=1, Cmd=cmdKey, Shift=shiftKey)
        RegisterEventHotKey(UInt32(1), UInt32(cmdKey | shiftKey), hotKeyID, GetApplicationEventTarget(), 0, &hotKey)
    }
    
    func startStdinListener() {
        DispatchQueue.global(qos: .userInitiated).async {
            while let line = readLine() {
                let trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)
                if !trimmed.isEmpty,
                   let cmdData = trimmed.data(using: .utf8) {
                    do {
                        let command = try JSONDecoder().decode(IPCCommand.self, from: cmdData)
                        self.viewModel.handleCommand(command)
                    } catch {
                        fputs("Swift JSON Error: \(error)\n", stderr)
                    }
                }
            }
        }
    }
    
    func windowWillClose(_ notification: Notification) {
        fputs("DEBUG: Window closing, reverting activation policy\n", stderr)
        NSApp.setActivationPolicy(.accessory)
    }
    
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return true
    }
}
