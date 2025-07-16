package main

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/aabiji/drip/p2p"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx    context.Context
	events chan p2p.Message

	finder *p2p.PeerFinder
	syncer p2p.FileSyncer
}

func NewApp() *App {
	// TODO: read from settings json file
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	fullpath := path.Join(home, "Downloads")
	syncer := p2p.NewFileSyncer(fullpath)

	events := make(chan p2p.Message, 25)

	finder := p2p.NewPeerFinder(true, events, &syncer)

	return &App{events: events, syncer: syncer, finder: &finder}
}

func createFrontendBindings(jsPath string) error {
	output, err := os.OpenFile(jsPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer output.Close()

	frontendEvents := []string{
		p2p.TRANSFER_STATE,
		p2p.PEERS_UPDATED,
	}

	for _, event := range frontendEvents {
		jsLine := fmt.Sprintf("export const %s = '%s';\n", event, event)
		_, err := output.WriteString(jsLine)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// launch the peer finder service
	go func() {
		if err := a.finder.Run(a.ctx); err != nil {
			panic(err)
		}
	}()

	// launch the event emitter service
	go func() {
		for event := range a.events {
			runtime.EventsEmit(a.ctx, event.MessageType, event)
		}
	}()

	// Generate frontend bindings for our event types
	err := createFrontendBindings("frontend/src/constants.js")
	if err != nil {
		panic(err)
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.syncer.Close()
	for _, peer := range a.finder.Peers {
		peer.Close()
	}
	close(a.events)
}

func (a *App) GetPeers() []string {
	names := []string{}
	for name, peer := range a.finder.Peers {
		if peer.Connected() {
			names = append(names, name)
		}
	}
	return names
}

func (a *App) StartFileTransfer(info p2p.TransferInfo) bool {
	msg := p2p.NewMessage(p2p.TRANSFER_INFO, info)
	for _, peerId := range info.Recipients {
		a.finder.Peers[peerId].Webrtc.QueueMessage(msg)
		a.syncer.SenderMarkTransfer(info)
	}
	return true
}

func (a *App) SendFileChunk(chunk p2p.FileChunk) bool {
	recipients, err := a.syncer.TransferRecipients(chunk.TransferId)
	if err != nil {
		panic(err) // TODO: tell the user
	}

	msg := p2p.NewMessage(p2p.TRANSFER_CHUNK, chunk)
	for _, peerId := range recipients {
		a.finder.Peers[peerId].Webrtc.QueueMessage(msg)
	}
	return true
}
