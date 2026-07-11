package main

import (
	"embed"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// version is overridden at build time via -ldflags "-X main.version=1.2.3"
// (see .github/workflows/release.yml). Defaults to a clearly-fake value for
// `wails dev`/unversioned local builds so the updater never mistakes a dev
// build for a real release needing an update.
var version = "0.0.0-dev"

func main() {
	// Velopack's Update.exe launches the freshly-installed exe with a
	// --veloapp-{install,updated,obsolete,uninstall} hook argument at various
	// points in the install/update lifecycle, expecting the process to do any
	// quick cleanup and exit within ~15s. The official SDKs (VelopackApp.Run(),
	// C#/Rust/C++/JS — there is no Go SDK) intercept this at the very top of
	// main() for exactly that reason. Without it, this arg falls straight
	// through into a full wails.Run(), so the hook invocation opens a real
	// visible window; Velopack then kills it once its timeout elapses and
	// launches the real restart on top, which reads as the app closing,
	// reopening, and closing again during every update.
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--veloapp-") {
			return
		}
	}

	// Create an instance of the app structure
	app := NewApp(version)

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "TicketSmith",
		Width:  1024,
		Height: 768,
		// Kept hidden until the frontend calls WindowReady (once React has
		// mounted its default screen) so the window never shows a stale
		// WebView2 frame or blank flash before the first real paint.
		StartHidden: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
