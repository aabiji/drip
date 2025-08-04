package main

import (
	"context"
	"fmt"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/x/explorer"
	"gioui.org/x/notify"

	"github.com/aabiji/drip/p2p"
)

type App struct {
	node     *p2p.Node
	ui       *UI
	settings Settings

	ctx    context.Context
	cancel context.CancelFunc
}

func NewApp() App {
	ctx, cancel := context.WithCancel(context.Background())
	a := App{
		settings: loadSettings(),
		ctx:      ctx,
		cancel:   cancel,
	}
	a.ui = NewUI(&a.settings, a.sendFiles)
	a.node = p2p.NewNode(ctx, &a.ui.settings.DownloadPath)
	return a
}

func (a *App) openWindow() {
	var ops op.Ops
	window := new(app.Window)
	a.ui.picker = explorer.NewExplorer(window)

	for {
		switch event := window.Event().(type) {
		case app.DestroyEvent:
			return
		case app.FrameEvent:
			gtx := app.NewContext(&ops, event)
			a.ui.DrawFrame(gtx)
			event.Frame(gtx.Ops)
		}
	}
}

func (a *App) Launch() {
	go func() { a.openWindow() }()
	app.Main()
}

func (a *App) Shutdown() {
	saveSettings(a.settings)
	a.cancel()
	a.node.Shutdown()
}

func (a App) sendFiles() {
	recipients := []string{}
	for _, peer := range a.ui.recipients {
		recipients = append(recipients, peer.name)
	}

	files := map[string]*p2p.File{}
	for _, file := range a.ui.files {
		f := p2p.NewReaderFile(file.name, file.size, file.rc)
		files[file.name] = f
	}

	a.node.SendFiles(recipients, files)
}

func (a App) askForAuth(peerId string) bool {
	fmt.Println("Asking the user to authorize a transfer from peerId...")
	return true
}

func (a App) notifyTransfer(peerId string, numFiles int) {
	notifier, _ := notify.NewNotifier()
	msg := fmt.Sprintf("Got %d files from %s\n", numFiles, peerId)
	_, _ = notifier.CreateNotification("Transfer status", msg)
}
