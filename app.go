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

	finder     *p2p.PeerFinder
	downloader p2p.Downloader
}

func NewApp() *App {
	// TODO: read from settings json file
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	fullpath := path.Join(home, "Downloads")
	downloader := p2p.NewDownloader(fullpath)

	events := make(chan p2p.Message, 25)

	finder := p2p.NewPeerFinder(true, events, &downloader)

	return &App{events: events, downloader: downloader, finder: &finder}
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
	a.downloader.Close()
	for _, peer := range a.finder.Peers {
		peer.Close()
	}
	close(a.events)
}

func (a *App) GetPeers() []string { return a.finder.GetConnectedPeers() }

func (a *App) StartFileTransfer(info p2p.TransferInfo) bool {
	msg := p2p.NewMessage(p2p.TRANSFER_INFO, info)
	for _, peerId := range info.Recipients {
		a.finder.Peers[peerId].Webrtc.QueueMessage(msg)
	}
	return true
}

func (a *App) SendFileChunk(chunk p2p.FileChunk) bool {
	msg := p2p.NewMessage(p2p.TRANSFER_CHUNK, chunk)
	for _, peerId := range chunk.Recipients {
		a.finder.Peers[peerId].Webrtc.QueueMessage(msg)
	}
	return true
}
