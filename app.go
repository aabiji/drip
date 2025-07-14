package main

import (
	"context"

	"github.com/aabiji/drip/p2p"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx    context.Context
	events chan string

	syncer p2p.FileSyncer
	finder p2p.PeerFinder
}

func NewApp() *App {
	syncer := p2p.NewFileSyncer()

	return &App{
		events: make(chan string, 10),
		syncer: syncer,
		finder: p2p.NewPeerFinder(true, &syncer),
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
	for name := range a.finder.Peers {
		if a.finder.Peers[name].Connected() {
			names = append(names, name)
		}
	}
	return names
}

func (a *App) StartFileTransfer(info p2p.FileInfo) bool {
	msg := p2p.NewMessage(p2p.TRANSFER_START, info)
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
