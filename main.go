package main

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

const (
	HOME_PAGE = iota
	SETTINGS_PAGE
	PROGRSSS_PAGE
)

type ItemState struct {
	value   any
	checked *widget.Bool // TODO: doesn't need to be a pointer
	clicked *widget.Clickable
}

type UI struct {
	peersList *widget.List
	filesList *widget.List
	// TODO: this is a horrible idea. instead, just assign to fields,
	//       since we probably won't end up having that many
	states   map[string]*ItemState
	page     int
	numFiles int
	numPeers int
}

type C = layout.Context
type D = layout.Dimensions
type T = *material.Theme

func main() {
	go func() {
		window := new(app.Window)
		var ops op.Ops

		theme := material.NewTheme()
		roboto, err := loadFont("roboto.ttf")
		if err != nil {
			panic(err)
		}
		theme.Shaper = text.NewShaper(text.WithCollection(roboto))

		ui := UI{
			peersList: &widget.List{List: layout.List{Axis: layout.Vertical}},
			filesList: &widget.List{List: layout.List{Axis: layout.Vertical}},
			numFiles:  5, numPeers: 5,
			states: make(map[string]*ItemState),
			page:   SETTINGS_PAGE,
		}
		for i := 0; i < 5; i++ {
			ui.states[fmt.Sprintf("PEER-%d", i)] = &ItemState{
				value:   fmt.Sprintf("Peer %d", i),
				checked: new(widget.Bool),
			}
			ui.states[fmt.Sprintf("FILE-%d", i)] = &ItemState{
				value:   fmt.Sprintf("File %d", i),
				clicked: new(widget.Clickable),
			}
		}

		for {
			switch event := window.Event().(type) {
			case app.DestroyEvent:
				os.Exit(0)
			case app.FrameEvent:
				gtx := app.NewContext(&ops, event)
				drawFrame(&ui, gtx, theme)
				event.Frame(gtx.Ops)
			}
		}

	}()
	app.Main()
}

func loadFont(path string) ([]font.FontFace, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	faces, err := opentype.ParseCollection(contents)
	return faces, err
}

// TODO: cache icons
func drawIcon(data []byte, size int, lightMode bool) func(C) D {
	return func(gtx C) D {
		icon, err := widget.NewIcon(data)
		if err != nil {
			panic(err)
		}

		c := color.NRGBA{A: 255}
		if !lightMode {
			c = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		}

		gtx.Constraints.Min.X = size
		gtx.Constraints.Min.Y = size
		return icon.Layout(gtx, c)
	}
}

func drawFileEntry(gtx C, theme T, state *ItemState) D {
	return layout.Flex{
		Alignment: layout.Start,
		Axis:      layout.Horizontal,
		Spacing:   layout.SpaceBetween,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			str := fmt.Sprintf("%v", state.value)
			return material.Label(theme, 16, str).Layout(gtx)
		}),

		layout.Rigid(func(gtx C) D {
			remove, err := widget.NewIcon(icons.NavigationClose)
			if err != nil {
				panic(err)
			}
			return material.IconButton(theme, state.clicked, remove, "Remove file").Layout(gtx)
		}),
	)
}

/*
TODO:
header, progressbar, list, div, files selection, notification
*/

func drawFrame(ui *UI, gtx C, theme T) {
	layout.Stack{}.Layout(gtx,
		// absolutely positioned icon
		layout.Stacked(func(gtx C) D {
			return layout.Inset{Top: 16, Left: 16}.Layout(gtx, func(gtx C) D {
				iconName := icons.ActionSettings
				if ui.page != HOME_PAGE {
					iconName = icons.NavigationArrowBack
				}
				// TODO: make this a clickable icon
				return layout.E.Layout(gtx, drawIcon(iconName, 64, true))
			})
		}),

		// page content, horizontally & vertically centered, 75% width, 80% height
		layout.Stacked(func(gtx C) D {
			return layout.Center.Layout(gtx, func(gtx C) D {
				width := gtx.Constraints.Max.X * 75 / 100
				xPadding := (gtx.Constraints.Max.X - width) / 4

				height := gtx.Constraints.Max.Y * 80 / 100
				yPadding := (gtx.Constraints.Max.Y - height) / 4

				return layout.Inset{
					Top:    unit.Dp(yPadding),
					Bottom: unit.Dp(yPadding),
					Left:   unit.Dp(xPadding),
					Right:  unit.Dp(xPadding),
				}.Layout(gtx, func(gtx C) D {
					gtx.Constraints = layout.Exact(image.Pt(width, height))

					if ui.page == HOME_PAGE {
						return drawHomePage(gtx, theme, ui)
					} else {
						return drawSettingsPage(gtx, theme, ui)
					}
				})
			})
		}),
	)
}

func drawHomePage(gtx C, theme T, ui *UI) D {
	return layout.Flex{
		Alignment: layout.Middle,
		Spacing:   layout.SpaceEvenly,
		Axis:      layout.Vertical,
	}.Layout(gtx,
		// list of peers
		layout.Flexed(0.5, func(gtx C) D {
			return material.List(theme, ui.peersList).Layout(gtx, ui.numPeers, func(gtx C, i int) D {
				item := ui.states[fmt.Sprintf("PEER-%d", i)]
				return material.CheckBox(theme, item.checked, fmt.Sprintf("%v", item.value)).Layout(gtx)
			})
		}),

		// file upload area
		layout.Rigid(func(gtx C) D {
			item := ui.states["UPLOAD"]
			fmt.Println(item.clicked.Clicked(gtx))

			return item.clicked.Layout(gtx, func(gtx C) D {
				pointer.CursorPointer.Add(gtx.Ops)

				return layout.Flex{Alignment: layout.Start, Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(drawIcon(icons.FileFileUpload, 100, true)),
					layout.Rigid(func(gtx C) D { return material.Label(theme, 20, "Select files").Layout(gtx) }),
				)
			})
		}),

		// list of selected files
		layout.Flexed(0.5, func(gtx C) D {
			return material.List(theme, ui.filesList).Layout(gtx, ui.numFiles, func(gtx C, i int) D {
				return drawFileEntry(gtx, theme, ui.states[fmt.Sprintf("FILE-%d", i)])
			})
		}),

		// send button
		layout.Rigid(func(gtx C) D {
			return material.Button(theme, ui.states["SEND-BTN"].clicked, "Send files").Layout(gtx)
		}),
	)
}

// theme, trust peers, show notifications, download folder, copyright

func drawSettingsPage(gtx C, theme T, ui *UI) D {
	optionElements := []layout.FlexChild{}
	options := []string{"Trust peers", "Show notifications"}
	for _, option := range options {
		optionElements = append(optionElements, layout.Rigid(func(gtx C) D {
			return material.CheckBox(theme, ui.states[option].checked, option).Layout(gtx)
		}))
	}

	return layout.Flex{
		Alignment: layout.Middle,
		Spacing:   layout.SpaceEvenly,
		Axis:      layout.Vertical,
	}.Layout(gtx,
		optionElements...,
	)
}
