package main

import (
	"github.com/aabiji/drip/p2p"
)

func main() {
	events := make(chan int, 10)
	discovery := p2p.NewPeerDiscovery()
	if err := discovery.Run(events); err != nil {
		panic(err)
	}
}
