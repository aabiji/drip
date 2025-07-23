package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

/*
- Make transfers significantly faster
- Handle errors in the backend
- Add app logging to file
- Completely efactor the backend implementation
- Port to android
- Fix bugs
*/

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "drip",
		Width:             900,
		Height:            700,
		AssetServer:       &assetserver.Options{Assets: assets},
		OnStartup:         app.startup,
		OnShutdown:        app.shutdown,
		Bind:              []any{app},
		HideWindowOnClose: false,
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
