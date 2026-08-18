package main

import (
	"embed"
	"encoding/json"
	"os"
	"runtime"

	"flacidal/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsJSON []byte

var appVersion = func() string {
	var cfg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(wailsJSON, &cfg); err != nil || cfg.Version == "" {
		return "dev"
	}
	return cfg.Version
}()

func init() {
	// Work around a WebKit/JSC signal handler clash on Linux that otherwise
	// crashes with SIGSEGV: JavaScriptCore defaults to SIGUSR1 (signal 10) for
	// its GC, which steps on Go's own signal handling, so point it at SIGUSR2
	// (signal 12) instead.
	if runtime.GOOS == "linux" {
		os.Setenv("JSC_SIGNAL_FOR_GC", "12")
	}
}

func main() {
	flacidalApp := app.NewApp(appVersion)

	err := wails.Run(&options.App{
		Title:  "FLACidal",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 10, G: 10, B: 10, A: 1},
		DragAndDrop:      &options.DragAndDrop{EnableFileDrop: true},
		OnStartup:        flacidalApp.Startup,
		OnShutdown:       flacidalApp.Shutdown,
		Bind: []interface{}{
			flacidalApp,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
