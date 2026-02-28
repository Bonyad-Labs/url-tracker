// Package ui provides native macOS UI components using AppleScript and systray.
package ui

import (
	"github.com/getlantern/systray"
)

// MenuHandlers defines the callbacks for various menu item actions.
type MenuHandlers struct {
	OnWhitelist       func()                       // Triggered when "Add to Whitelist" is clicked
	OnManageWhitelist func()                       // Triggered when "Manage Whitelist" is clicked
	OnPreferences     func()                       // Triggered when "Preferences..." is clicked
	OnSearch          func()                       // Triggered when "Search Saved URLs" is clicked
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
		mSearch := systray.AddMenuItem("Search Saved URLs", "Open interactive search")
		systray.AddSeparator()
		mPause := systray.AddMenuItem("Pause Monitoring", "Temporarily stop tracking new URLs")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the application")

		go func() {
			for {
				select {
				case <-mWhitelist.ClickedCh:
					handlers.OnWhitelist()
				case <-mManage.ClickedCh:
					handlers.OnManageWhitelist()
				case <-mPreferences.ClickedCh:
					handlers.OnPreferences()
				case <-mSearch.ClickedCh:
					handlers.OnSearch()
				case <-mPause.ClickedCh:
					handlers.OnTogglePause(mPause)
				case <-mQuit.ClickedCh:
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
