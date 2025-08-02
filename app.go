package main

import (
	"context"
	"fmt"

	"gioui.org/x/explorer"
	"gioui.org/x/notify"

	"github.com/aabiji/drip/p2p"
)

type App struct {
	node *p2p.Node
	ui   *UI

	ctx    context.Context
	cancel context.CancelFunc
}

func NewApp(e *explorer.Explorer) App {
	ctx, cancel := context.WithCancel(context.Background())

	app := App{
		ui:  NewUI(e),
		ctx: ctx, cancel: cancel,
	}
	app.node = p2p.NewNode(ctx, &app.ui.settings.DownloadPath,
		app.askForAuth, app.notifyTransfer)

	return app
}

func (app *App) Shutdown() {
	saveSettings(app.ui.settings)
	app.cancel()
	app.node.Shutdown()
}

func (app App) askForAuth(peerId string) bool {
	fmt.Println("Asking the user to authorize a transfer from peerId...")
	return true
}

func (app App) notifyTransfer(peerId string, numFiles int) {
	notifier, _ := notify.NewNotifier()
	msg := fmt.Sprintf("Got %d files from %s\n", numFiles, peerId)
	_, _ = notifier.CreateNotification("Transfer status", msg)
}

/*
	_, err = readCloser.Read(entry.data)
	if err != nil {
		continue
	}
	readCloser.Close()
*/
