//go:build windows

package main

import (
	"log"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.design/x/hotkey"
)

// setupQuickCaptureHotkey registers the system-wide quick-capture shortcut
// (Ctrl+T) and, on each press, raises the window and tells the frontend
// to open the quick-capture dialog (see "quickcapture:open" in App.tsx).
// Returns the human-readable shortcut label on success, or "" if
// registration failed — the frontend uses that to decide whether to
// advertise the shortcut at all (see QuickCaptureShortcut in app.go).
//
// Windows-only: on Windows golang.design/x/hotkey runs each hotkey on its own
// OS thread internally, so registering here (off the main thread) is safe. On
// macOS the package needs the process main thread (mainthread.Init), which
// Wails already owns for its run loop — quickcapture_other.go stubs this out
// there.
//
// Registration failure (the combination is already taken by another app) is
// logged and otherwise ignored: the app works fine without the shortcut.
func (a *App) setupQuickCaptureHotkey() string {
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyT)
	if err := hk.Register(); err != nil {
		log.Printf("ticketsmith: register quick-capture hotkey: %v", err)
		return ""
	}
	go func() {
		for range hk.Keydown() {
			wailsruntime.WindowUnminimise(a.ctx)
			wailsruntime.WindowShow(a.ctx)
			wailsruntime.EventsEmit(a.ctx, "quickcapture:open")
		}
	}()
	return "Ctrl+T"
}
