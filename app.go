package main

import (
	"context"
	"fmt"

	"github.com/aabiji/drip/p2p"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx    context.Context
	events chan p2p.Message

	finder     *p2p.PeerFinder
	downloader *p2p.Downloader
}

func NewApp() *App {
	downloader := p2p.NewDownloader()
	events := make(chan p2p.Message, 25)
	finder := p2p.NewPeerFinder(true, events)

	return &App{
		events:     events,
		downloader: downloader,
		finder:     &finder,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// launch the peer finder service
	go func() {
		if err := a.finder.Run(a.ctx, a.downloader); err != nil {
			panic(err)
		}
	}()

	// launch the event emitter service
	go func() {
		for event := range a.events {
			runtime.EventsEmit(a.ctx, event.MessageType, event)
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	a.downloader.Close()
	for _, peer := range a.finder.Peers {
		peer.Close()
	}
	close(a.events)
}

func (a *App) GetPeers() []string { return a.finder.GetConnectedPeers() }

// the following functions are exported to the frontend
func (a *App) RequestSessionAuth(info p2p.SessionInfo) error {
	msg := p2p.NewMessage(p2p.SESSION_INFO, info)
	for _, peerId := range info.Recipients {
		peer, exists := a.finder.Peers[peerId]
		if exists && peer.Connected() {
			peer.Webrtc.QueueMessage(msg)
		}
	}
	return nil
}

func (a *App) SendFileChunk(chunk p2p.TransferChunk) error {
	peer, exists := a.finder.Peers[chunk.Recipient]
	if !exists || !peer.Connected() {
		return fmt.Errorf("%s has disconnected", chunk.Recipient)
	}

	msg := p2p.NewMessage(p2p.TRANSFER_CHUNK, chunk)
	a.finder.Peers[chunk.Recipient].Webrtc.QueueMessage(msg)
	return nil
}

func (a *App) CancelSession(signal p2p.SessionCancel) error {
	msg := p2p.NewMessage(p2p.SESSION_CANCEL, signal)
	for _, peerId := range signal.Recipients {
		peer, exists := a.finder.Peers[peerId]
		if exists {
			peer.Webrtc.QueueMessage(msg)
		}
	}
	return nil
}
