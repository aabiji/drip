package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aabiji/drip/p2p"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx    context.Context
	events chan p2p.Message

	finder       *p2p.PeerFinder
	downloader   p2p.Downloader
	settingsPath string
}

func NewApp() *App {
	settingsPath := "drip-state.json"
	downloader, err := p2p.NewDownloader(settingsPath)
	if err != nil {
		panic(err)
	}
	events := make(chan p2p.Message, 25)
	finder := p2p.NewPeerFinder(true, events, &downloader)

	return &App{
		events:       events,
		downloader:   downloader,
		finder:       &finder,
		settingsPath: settingsPath,
	}
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
	err := a.downloader.Close(a.settingsPath)
	if err != nil {
		panic(err)
	}
	for _, peer := range a.finder.Peers {
		peer.Close()
	}
	close(a.events)
}

func (a *App) GetPeers() []string { return a.finder.GetConnectedPeers() }

// the following functions are exported to the frontend
func (a *App) StartFileTransfer(info p2p.TransferInfo) {
	msg := p2p.NewMessage(p2p.TRANSFER_INFO, info)
	a.finder.Peers[info.Recipient].Webrtc.QueueMessage(msg)
}

func (a *App) SendFileChunk(chunk p2p.FileChunk) {
	msg := p2p.NewMessage(p2p.TRANSFER_CHUNK, chunk)
	a.finder.Peers[chunk.Recipient].Webrtc.QueueMessage(msg)
}
