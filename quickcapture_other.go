//go:build !windows

package main

// The quick-capture global hotkey is Windows-only for now:
// golang.design/x/hotkey requires the process main thread on macOS
// (mainthread.Init), which Wails already owns for its own run loop.
// See quickcapture_windows.go for the real implementation. Returning ""
// tells the frontend not to advertise any shortcut.
func (a *App) setupQuickCaptureHotkey() string { return "" }
