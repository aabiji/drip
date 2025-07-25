package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// icons from: https://www.freepik.com/author/stockio/icons/stockio-fill_597

func main() {
	a := app.New()
	w := a.NewWindow("Drip")

	label := widget.NewLabel("hello world!")

	checkbox := widget.NewCheck("Select me!", func(changed bool) {
		fmt.Println("Checked: ", changed)
	})

	button := widget.NewButton("Click me!", func() {
		dialog.ShowFileOpen(func(file fyne.URIReadCloser, err error) {
			if err != nil {
				panic(err)
			}
			if file != nil {
				fmt.Println(file.URI().Path())
			}
		}, w)
	})

	bar := widget.NewProgressBar()
	bar.Min = 0
	bar.Max = 100
	bar.Value = 50

	values := []string{"Value #1", "Value #2", "Value #3", "Value #4", "Value #5"}
	list := widget.NewList(
		func() int {
			return len(values)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(values[i])
		},
	)
	scroll := container.NewScroll(list)
	scroll.SetMinSize(fyne.NewSize(500, 300))

	res, err := fyne.LoadResourceFromPath("icons/upload.png")
	if err != nil {
		panic(err)
	}
	icon := widget.NewIcon(res)

	box := container.NewVBox(label, checkbox, button, bar, scroll, icon)

	// system tray menu, drag and drop

	w.SetContent(box)
	w.Resize(fyne.NewSize(700, 500))
	w.ShowAndRun()
}
