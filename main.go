package main

import (
	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/x/explorer"
	"os"
)

func main() {
	go func() {
		var ops op.Ops
		window := new(app.Window)
		e := explorer.NewExplorer(window)
		a := NewApp(e)

		for {
			switch event := window.Event().(type) {
			case app.DestroyEvent:
				a.Shutdown()
				os.Exit(0)
			case app.FrameEvent:
				gtx := app.NewContext(&ops, event)
				a.ui.DrawFrame(gtx)
				event.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}
