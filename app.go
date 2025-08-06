package main

import (
	"context"
	"time"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/x/explorer"
	"gioui.org/x/notify"

	"github.com/aabiji/drip/p2p"
)

type App struct {
	node       *p2p.Node
	ui         *UI
	settings   Settings
	appEvents  chan p2p.Message
	nodeEvents chan p2p.Message

	// TODO: we should queue this
	// for example, if multiple peers attempt to send us files in successoin
	currentResponse p2p.TransferResponse
	requestSender   string
	currentTransfer string

	ctx    context.Context
	cancel context.CancelFunc
}

func NewApp() App {
	ctx, cancel := context.WithCancel(context.Background())
	a := App{
		settings:   loadSettings(),
		appEvents:  make(chan p2p.Message),
		nodeEvents: make(chan p2p.Message),
		ctx:        ctx,
		cancel:     cancel,
	}
	a.ui = NewUI(&a.settings, a.appEvents)
	a.node = p2p.NewNode(ctx, &a.ui.settings.DownloadPath,
		a.appEvents, a.nodeEvents)
	go a.handleAppEvents()
	return a
}

func (a *App) Launch() {
	go func() { a.openWindow() }()
	app.Main()
}

func (a *App) Shutdown() {
	saveSettings(a.settings)
	close(a.appEvents)
	close(a.nodeEvents)
	a.cancel()
	a.node.Shutdown()
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

func (a *App) sendFiles() {
	recipients := []string{}
	for _, peer := range a.ui.recipients {
		if peer.check.Value {
			recipients = append(recipients, peer.name)
		}
	}

	files := map[string]*p2p.File{}
	for _, file := range a.ui.files {
		f := p2p.NewReaderFile(file.name, file.size, file.rc)
		files[file.name] = f
	}

	a.currentTransfer = a.node.SendFiles(recipients, files)
	a.ui.currentPage = PROGRESS_PAGE
	go func() {
		for a.ui.currentPage == PROGRESS_PAGE {
			percentages := a.node.GetFilePercentages(a.currentTransfer)
			a.ui.UpdateFileProgresses(percentages)
			time.Sleep(time.Second)
		}
	}()
}

func (a *App) handleAppEvents() {
	for event := range a.appEvents {
		switch event.Type {
		case p2p.SEND_FILES:
			a.sendFiles()

		case p2p.ADDED_PEER, p2p.REMOVED_PEER:
			peer, err := p2p.Deserialize[string](event)
			if err != nil {
				panic(err)
			}
			a.ui.UpdateRecipients(peer, event.Type == p2p.REMOVED_PEER)

		case p2p.NOTIFY_COMPLETION:
			msg, err := p2p.Deserialize[string](event)
			if err != nil {
				panic(err)
			}
			notifier, _ := notify.NewNotifier()
			_, _ = notifier.CreateNotification("Transfer status", msg)

		case p2p.TRANSFER_REQUEST:
			// show the request to the user
			request, err := p2p.Deserialize[p2p.TransferRequest](event)
			if err != nil {
				panic(err)
			}
			a.currentResponse = p2p.TransferResponse{TransferId: request.TransferId}
			a.requestSender = request.Sender
			a.ui.showAuthPopup = true
			a.ui.authMsg = request.Message

		case p2p.AUTH_GRANTED:
			// relay back the user's choice
			authorized, err := p2p.Deserialize[bool](event)
			if err != nil {
				panic(err)
			}

			a.ui.showAuthPopup = false
			a.currentResponse.Authorized = authorized
			msg := p2p.NewMessage(p2p.TRANSFER_RESPONSE, a.currentResponse)
			msg.Recipients = []string{a.requestSender}
			a.nodeEvents <- msg
		}
	}
}
