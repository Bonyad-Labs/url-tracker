// Package ui provides native macOS UI components using AppleScript and systray.
package ui

import (
	"log"

	"github.com/getlantern/systray"
)

// MenuHandlers defines the callbacks for various menu item actions.
type MenuHandlers struct {
	OnManageWhitelist func() // Triggered when "Manage Whitelist" is clicked
	OnPreferences     func() // Triggered when "Preferences..." is clicked
	OnManageBookmarks func() // Triggered when "Manage Bookmarks" is clicked
	OnSaveTab         func() // Triggered when "Save Current Tab" is clicked
	OnQuit            func() // Triggered when "Quit" is clicked
}

// StartMenu initializes and runs the system tray menu.
// On macOS, this MUST be called from the main thread and is a blocking call.
func StartMenu(handlers MenuHandlers) {
	onReady := func() {
		systray.SetTemplateIcon(GetAppIcon(), GetAppIcon())
		systray.SetTooltip("Monitoring Chrome URLs")

		mManage := systray.AddMenuItem("Manage Whitelist", "View or remove whitelisted items")
		mPreferences := systray.AddMenuItem("Preferences...", "Change application settings and configuration")
		systray.AddSeparator()
		mSaveTab := systray.AddMenuItem("Save Current Tab", "Instantly capture the active browser tab")
		mManageBookmarks := systray.AddMenuItem("Manage Bookmarks", "Search, Edit, or Remove saved URLs")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the application")

		go func() {
			for {
				select {
				case <-mManage.ClickedCh:
					log.Printf("Menu: Manage Whitelist clicked")
					handlers.OnManageWhitelist()
				case <-mPreferences.ClickedCh:
					log.Printf("Menu: Preferences clicked")
					handlers.OnPreferences()
				case <-mManageBookmarks.ClickedCh:
					log.Printf("Menu: Manage Bookmarks clicked")
					handlers.OnManageBookmarks()
				case <-mSaveTab.ClickedCh:
					log.Printf("Menu: Save Current Tab clicked")
					handlers.OnSaveTab()
				case <-mQuit.ClickedCh:
					log.Printf("Menu: Quit clicked")
					handlers.OnQuit()
				}
			}
		}()
	}

	onExit := func() {
		// Clean up here if needed
	}

	systray.Run(onReady, onExit)
}
