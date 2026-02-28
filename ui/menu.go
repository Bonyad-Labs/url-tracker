// Package ui provides native macOS UI components using AppleScript and systray.
package ui

import (
	"log"

	"github.com/getlantern/systray"
)

// MenuHandlers defines the callbacks for various menu item actions.
type MenuHandlers struct {
	OnWhitelist       func()                       // Triggered when "Add to Whitelist" is clicked
	OnManageWhitelist func()                       // Triggered when "Manage Whitelist" is clicked
	OnPreferences     func()                       // Triggered when "Preferences..." is clicked
	OnManageBookmarks func()                       // Triggered when "Manage Bookmarks" is clicked
	OnTogglePause     func(item *systray.MenuItem) // Triggered when "Pause Monitoring" is clicked
	OnQuit            func()                       // Triggered when "Quit" is clicked
}

// StartMenu initializes and runs the system tray menu.
// On macOS, this MUST be called from the main thread and is a blocking call.
func StartMenu(handlers MenuHandlers) {
	onReady := func() {
		systray.SetTemplateIcon(GetAppIcon(), GetAppIcon())
		systray.SetTooltip("Monitoring Chrome URLs")

		mWhitelist := systray.AddMenuItem("Add Domain to Whitelist", "Exclude a domain from monitoring")
		mManage := systray.AddMenuItem("Manage Whitelist", "View or remove whitelisted items")
		mPreferences := systray.AddMenuItem("Preferences...", "Change application settings and configuration")
		systray.AddSeparator()
		mManageBookmarks := systray.AddMenuItem("Manage Bookmarks", "Search, Edit, or Remove saved URLs")
		systray.AddSeparator()
		mPause := systray.AddMenuItem("Pause Monitoring", "Temporarily stop tracking new URLs")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the application")

		go func() {
			for {
				select {
				case <-mWhitelist.ClickedCh:
					log.Printf("Menu: Add Domain to Whitelist clicked")
					handlers.OnWhitelist()
				case <-mManage.ClickedCh:
					log.Printf("Menu: Manage Whitelist clicked")
					handlers.OnManageWhitelist()
				case <-mPreferences.ClickedCh:
					log.Printf("Menu: Preferences clicked")
					handlers.OnPreferences()
				case <-mManageBookmarks.ClickedCh:
					log.Printf("Menu: Manage Bookmarks clicked")
					handlers.OnManageBookmarks()
				case <-mPause.ClickedCh:
					log.Printf("Menu: Toggle Pause clicked")
					handlers.OnTogglePause(mPause)
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
