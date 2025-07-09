package main

import (
	"context"
	"fmt"

	"github.com/aabiji/drip/p2p"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	events chan string
	finder p2p.PeerFinder
}

func NewApp() *App {
	return &App{
		events: make(chan string, 10),
		finder: p2p.NewPeerFinder(true),
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

func (a *App) StartFileTransfer(info p2p.TransferInfo) bool {
	fmt.Println("starting file transfer", info)
	return true
}

func (a *App) SendFileChunk(chunk p2p.FileChunk) bool {
	fmt.Println("getting chunk", chunk)
	return true
}
