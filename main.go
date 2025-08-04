package main

import (
	"os"
	"os/signal"
)

func main() {
	app := NewApp()
	app.Launch()

	// handle ctrl-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		app.Shutdown()
	}()
}
