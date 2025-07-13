package main

import (
	"context"

	"github.com/aabiji/drip/p2p"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	events          chan string
	transferService p2p.TransferService
	finder          p2p.PeerFinder
}

func NewApp() *App {
	t := p2p.NewTransferService()

	return &App{
		events:          make(chan string, 10),
		transferService: t,
		finder:          p2p.NewPeerFinder(true, &t),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// launch the peer finder service
	go func() {
		if err := a.finder.Run(a.events, a.ctx); err != nil {
			panic(err)
		}
	}()

	// launch the event emitter service
	go func() {
		for event := range a.events {
			runtime.EventsEmit(a.ctx, event)
		}
	}()
}

func (a *App) GetPeers() []string {
	names := []string{}
	for name, _ := range a.finder.Peers {
		names = append(names, name)
	}
	return names
}

func (a *App) StartFileTransfer(info p2p.TransferInfo) {
	msg := p2p.NewMessage(p2p.TRANSFER_START, info)

	for _, peerId := range info.Recipients {
		exists := a.finder.ConnectToPeer(peerId)
		if exists {
			a.transferService.Pending[peerId] <- msg
		}
	}
}

func (a *App) SendFileChunk(chunk p2p.FileChunk) {
	msg := p2p.NewMessage(p2p.TRANSFER_CHUNK, chunk)

	for _, peerId := range chunk.Recipients {
		exists := a.finder.ConnectToPeer(peerId)
		if exists {
			a.transferService.Pending[peerId] <- msg
		}
	}
}
