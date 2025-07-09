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
	go func() {
		if err := a.finder.Run(a.events, a.ctx); err != nil {
			panic(err)
		}
	}()
	go func() {
		for event := range a.events {
			fmt.Println(a.GetPeers())
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
