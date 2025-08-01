package main

import (
	"fmt"
	"gioui.org/x/explorer"
	"github.com/aabiji/drip/p2p"
)

type Settings struct {
	downloadFolder string
	trustPeers     bool
	notifyUser     bool
	lightmode      bool
}

type App struct {
	node     *p2p.Node
	settings Settings
	ui       UI
}

func NewApp(e *explorer.Explorer) App {
	settings := Settings{}
	app := App{settings: settings, ui: *NewUI(e)}
	app.node = p2p.NewNode(
		&settings.downloadFolder, app.askForAuth, app.notifyTransfer)
	return app
}

func (app App) askForAuth(peerId string) bool {
	fmt.Println("Asking the user to authorize a transfer from peerId...")
	return true
}

func (app App) notifyTransfer(peerId string, numFiles int) {
	fmt.Printf("Got %d files from %s\n", numFiles, peerId)
}
