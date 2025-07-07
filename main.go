package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/aabiji/drip/p2p"
)

func runP2P() {
	events := make(chan int, 10)
	discovery := p2p.NewPeerDiscovery()
	if err := discovery.Run(events); err != nil {
		panic(err)
	}
}

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:       "drip",
		Width:       900,
		Height:      700,
		AssetServer: &assetserver.Options{Assets: assets},
		OnStartup:   app.startup,
		Bind:        []any{app},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
