package main

import (
	"fmt"
	"context"
	"gioui.org/x/explorer"
	"github.com/aabiji/drip/p2p"
)

type App struct {
	node     *p2p.Node
	ui       *UI
	settings Settings

	ctx    context.Context
	cancel context.CancelFunc
}

func NewApp(e *explorer.Explorer) App {
	settings := loadSettings()
	ctx, cancel := context.WithCancel(context.Background())

	app := App{
		settings: settings,
		ui: NewUI(e),
		ctx: ctx, cancel: cancel,
	}
	app.node = p2p.NewNode(ctx, &settings.DownloadFolder,
	app.askForAuth, app.notifyTransfer)
	return app
}

// TODO: run on close
func (app *App) Shutdown() {
	saveSettings(app.settings)
	app.cancel()
	app.node.Shutdown()
}

func (app App) askForAuth(peerId string) bool {
	fmt.Println("Asking the user to authorize a transfer from peerId...")
	return true
}

func (app App) notifyTransfer(peerId string, numFiles int) {
	fmt.Printf("Got %d files from %s\n", numFiles, peerId)
}
