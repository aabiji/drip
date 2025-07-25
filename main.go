package main

import (
	"fmt"
	"image/color"
	"os"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

const (
	HOME_PAGE = iota
	SETTINGS_PAGE
	PROGRSSS_PAGE
)

type ListItem struct {
	value   any
	checked *widget.Bool
	clicked *widget.Clickable
}

type UI struct {
	peers     []ListItem
	peersList *widget.List

	files     []ListItem
	filesList *widget.List

	sendButton *widget.Clickable
	uploadArea *widget.Clickable

	page int
}

type C = layout.Context
type D = layout.Dimensions

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
			peersList: &widget.List{
				List: layout.List{Axis: layout.Vertical},
			},
			filesList: &widget.List{
				List: layout.List{Axis: layout.Vertical},
			},
			sendButton: new(widget.Clickable),
			uploadArea: new(widget.Clickable),
			page:       HOME_PAGE,
		}
		for i := 0; i < 5; i++ {
			ui.peers = append(ui.peers, ListItem{
				value:   fmt.Sprintf("Peer %d", i),
				checked: new(widget.Bool),
			})
			ui.files = append(ui.files, ListItem{
				value:   fmt.Sprintf("File %d", i),
				clicked: new(widget.Clickable),
			})
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

func drawFileEntry(gtx C, theme *material.Theme, item ListItem) D {
	return layout.Flex{
		Alignment: layout.Start,
		Axis:      layout.Horizontal,
		Spacing:   layout.SpaceBetween,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			str := fmt.Sprintf("%v", item.value)
			return material.Label(theme, 16, str).Layout(gtx)
		}),

		layout.Rigid(func(gtx C) D {
			remove, err := widget.NewIcon(icons.NavigationClose)
			if err != nil {
				panic(err)
			}
			return material.IconButton(theme, item.clicked, remove, "Remove file").Layout(gtx)
		}),
	)
}

/*
TODO:
header, progressbar, list, div, files selection, notification
make it look half decent
*/

func drawFrame(ui *UI, gtx C, theme *material.Theme) {
	iconName := icons.ActionSettings
	if ui.page != HOME_PAGE {
		iconName = icons.NavigationArrowBack
	}

	layout.Stack{}.Layout(gtx,
		// absolutely positioned icon
		layout.Stacked(func(gtx C) D {
			return layout.Inset{Top: 16, Right: 16}.Layout(gtx, func(gtx C) D {
				return layout.E.Layout(gtx, drawIcon(iconName, 64, true))
			})
		}),

		// page content
		layout.Stacked(func(gtx C) D {
			return layout.Center.Layout(gtx, func(gtx C) D {
				// 80% width
				maxWidth := gtx.Constraints.Max.X * 80 / 100
				gtx.Constraints.Max.X = maxWidth
				gtx.Constraints.Min.X = maxWidth

				return layout.Flex{
					Alignment: layout.Middle,
					Spacing:   layout.SpaceEvenly,
					Axis:      layout.Vertical,
				}.Layout(gtx,
					// list of peers
					layout.Flexed(0.5, func(gtx C) D {
						return material.List(theme, ui.peersList).Layout(gtx, len(ui.peers), func(gtx C, i int) D {
							str := fmt.Sprintf("%v", ui.peers[i].value)
							return material.CheckBox(theme, ui.peers[i].checked, str).Layout(gtx)
						})
					}),

					// file upload area
					layout.Rigid(func(gtx C) D {
						fmt.Println(ui.uploadArea.Clicked(gtx))

						return ui.uploadArea.Layout(gtx, func(gtx C) D {
							return layout.Flex{Alignment: layout.Start, Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(drawIcon(icons.FileFileUpload, 100, true)),
								layout.Rigid(func(gtx C) D { return material.Label(theme, 20, "Select files").Layout(gtx) }),
							)
						})
					}),

					// list of selected files
					layout.Flexed(0.5, func(gtx C) D {
						return material.List(theme, ui.filesList).Layout(gtx, len(ui.files), func(gtx C, i int) D {
							return drawFileEntry(gtx, theme, ui.files[i])
						})
					}),

					// send button
					layout.Rigid(func(gtx C) D {
						return material.Button(theme, ui.sendButton, "Send files").Layout(gtx)
					}),
				)
			})
		}),
	)

}
