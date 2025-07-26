package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"time"

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

const ( // pages
	HOME_PAGE = iota
	SETTINGS_PAGE
	PROGRESS_PAGE
)

const ( // icons
	BACK_ICON = iota
	GEAR_ICON
	UPLOAD_ICON
	CLOSE_ICON
)

const ( // buttons
	BTNS_START = iota
	UPLOAD_FILES
	SEND_FILES
	TOGGLE_THEME
	PICK_PATH
	PAGE_SWITCHER
	BTNS_END
)

type FileEntry struct {
	name     string
	clicked  widget.Clickable
	progress float32
}

type UI struct {
	peersList  *widget.List
	peerChecks []widget.Bool

	filesList *widget.List
	files     []FileEntry

	errorClicks []widget.Clickable
	errors      []error

	trustOption  widget.Bool
	notifyOption widget.Bool
	isLightMode  widget.Bool

	icons   []*widget.Icon
	buttons []widget.Clickable

	downloadPath string
	currentPage  int
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

		ui := initUI()

		for {
			switch event := window.Event().(type) {
			case app.DestroyEvent:
				os.Exit(0)
			case app.FrameEvent:
				gtx := app.NewContext(&ops, event)
				drawFrame(ui, gtx, theme)
				event.Frame(gtx.Ops)
			}
		}

	}()
	app.Main()
}

func initUI() *UI {
	ui := &UI{
		peersList:    &widget.List{List: layout.List{Axis: layout.Vertical}},
		filesList:    &widget.List{List: layout.List{Axis: layout.Vertical}},
		downloadPath: "~/Downloads",
		currentPage:  PROGRESS_PAGE,
	}

	iconData := [][]byte{
		icons.NavigationArrowBack, icons.ActionSettings,
		icons.FileFileUpload, icons.NavigationClose}
	for _, data := range iconData {
		icon, err := widget.NewIcon(data)
		if err != nil {
			panic(err)
		}
		ui.icons = append(ui.icons, icon)
	}

	ui.addError(errors.New("error #1"))
	ui.addError(errors.New("error #2"))

	for i := 0; i < BTNS_END-BTNS_START; i++ {
		ui.buttons = append(ui.buttons, widget.Clickable{})
	}

	for i := 0; i < 5; i++ {
		ui.peerChecks = append(ui.peerChecks, widget.Bool{})
		ui.files = append(ui.files, FileEntry{
			name:     fmt.Sprintf("File %d", i),
			progress: -1.0,
		})
	}

	return ui
}

func (ui *UI) switchPages(gtx C) {
	if ui.buttons[PAGE_SWITCHER].Clicked(gtx) {
		if ui.currentPage == HOME_PAGE {
			ui.currentPage = SETTINGS_PAGE
		} else if ui.currentPage == SETTINGS_PAGE {
			ui.currentPage = HOME_PAGE
		} else if ui.currentPage == PROGRESS_PAGE {
			ui.currentPage = HOME_PAGE
			// TODO: cancel sending...
		}
	}
}

func (ui *UI) addError(err error) {
	ui.errors = append(ui.errors, err)
	ui.errorClicks = append(ui.errorClicks, widget.Clickable{})
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

/*
TODO:
files selection, notification
*/

func drawFrame(ui *UI, gtx C, theme T) {
	layout.Stack{}.Layout(gtx,
		// absolutely positioned icon
		layout.Stacked(func(gtx C) D {
			return layout.Inset{Top: 16, Left: 16}.Layout(gtx, func(gtx C) D {
				ui.switchPages(gtx)
				i := ui.icons[GEAR_ICON]
				if ui.currentPage != HOME_PAGE {
					i = ui.icons[BACK_ICON]
				}
				return layout.E.Layout(gtx, func(gtx C) D {
					return material.IconButton(theme, &ui.buttons[PAGE_SWITCHER],
						i, "View settings").Layout(gtx)
				})
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

					if ui.currentPage == HOME_PAGE {
						return drawHomePage(gtx, theme, ui)
					} else if ui.currentPage == PROGRESS_PAGE {
						return drawProgressPage(gtx, theme, ui)
					} else {
						return drawSettingsPage(gtx, theme, ui)
					}
				})
			})
		}),

		// error tray
		layout.Stacked(func(gtx C) D {
			return layout.Center.Layout(gtx, func(gtx C) D {
				width := gtx.Constraints.Max.X * 30 / 100
				height := gtx.Constraints.Max.Y * 30 / 100

				return layout.Inset{}.Layout(gtx, func(gtx C) D {
					gtx.Constraints = layout.Exact(image.Pt(width, height))
					return drawErrorTray(gtx, theme, ui)
				})
			})
		}),
	)
}

func drawErrorTray(gtx C, theme T, ui *UI) D {
	widgets := []layout.FlexChild{}

	for i := range ui.errors {
		widgets = append(widgets,
			// toest notifications
			layout.Rigid(func(gtx C) D {
				return layout.Flex{
					Axis:      layout.Horizontal,
					Spacing:   layout.SpaceBetween,
					Alignment: layout.Middle,
				}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return material.Label(theme, 20,
							ui.errors[i].Error()).Layout(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						return material.IconButton(theme, &ui.errorClicks[i],
							ui.icons[CLOSE_ICON], "Dimiss error").Layout(gtx)
					}),
				)
			}))
	}

	return layout.Flex{
		Axis:      layout.Vertical,
		Spacing:   layout.SpaceStart,
		Alignment: layout.Middle,
	}.Layout(gtx, widgets...)
}

func drawFileEntry(gtx C, theme T, file *FileEntry) D {
	widgets := []layout.FlexChild{
		layout.Rigid(func(gtx C) D {
			return material.Label(theme, 16, file.name).Layout(gtx)
		}),
	}

	if file.progress < 0 {
		widgets = append(widgets,
			layout.Rigid(func(gtx C) D {
				remove, err := widget.NewIcon(icons.NavigationClose)
				if err != nil {
					panic(err)
				}
				return material.IconButton(theme, &file.clicked, remove, "Remove file").Layout(gtx)
			}))
	}

	entry := layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx, widgets...)

	if file.progress >= 0 {
		return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return entry }),
			layout.Rigid(func(gtx C) D {
				return material.ProgressBar(theme, 0.5).Layout(gtx)
			}))
	} else {
		return entry
	}
}

func drawHomePage(gtx C, theme T, ui *UI) D {
	return layout.Flex{
		Alignment: layout.Middle,
		Spacing:   layout.SpaceEvenly,
		Axis:      layout.Vertical,
	}.Layout(gtx,
		// list of peers
		layout.Flexed(0.5, func(gtx C) D {
			return material.List(theme, ui.peersList).Layout(gtx, len(ui.peerChecks), func(gtx C, i int) D {
				str := fmt.Sprintf("PEER-%d", i)
				return material.CheckBox(theme, &ui.peerChecks[i], str).Layout(gtx)
			})
		}),

		// file upload area
		layout.Rigid(func(gtx C) D {
			fmt.Println(ui.buttons[UPLOAD_FILES].Clicked(gtx))

			return ui.buttons[UPLOAD_FILES].Layout(gtx, func(gtx C) D {
				pointer.CursorPointer.Add(gtx.Ops)

				return layout.Flex{Alignment: layout.Start, Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(drawIcon(icons.FileFileUpload, 100, true)),
					layout.Rigid(func(gtx C) D { return material.Label(theme, 20, "Select files").Layout(gtx) }),
				)
			})
		}),

		// list of selected files
		layout.Flexed(0.5, func(gtx C) D {
			return material.List(theme, ui.filesList).Layout(gtx, len(ui.files), func(gtx C, i int) D {
				return drawFileEntry(gtx, theme, &ui.files[i])
			})
		}),

		// send button
		layout.Rigid(func(gtx C) D {
			return material.Button(theme, &ui.buttons[SEND_FILES], "Send files").Layout(gtx)
		}),
	)
}

func drawProgressPage(gtx C, theme T, ui *UI) D {
	return layout.Flex{
		Alignment: layout.Start,
		Axis:      layout.Vertical,
		Spacing:   layout.SpaceEvenly,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return material.H1(theme, "Sending files").Layout(gtx)
		}),

		layout.Flexed(0.5, func(gtx C) D {
			return material.List(theme, ui.filesList).Layout(gtx, len(ui.files), func(gtx C, i int) D {
				return drawFileEntry(gtx, theme, &ui.files[i])
			})
		}),
	)
}

func drawSettingsPage(gtx C, theme T, ui *UI) D {
	return layout.Flex{
		Alignment: layout.Start,
		Spacing:   layout.SpaceStart,
		Axis:      layout.Vertical,
	}.Layout(gtx,
		// toggle theme
		layout.Rigid(func(gtx C) D {
			return layout.Flex{
				Spacing: layout.SpaceBetween, Axis: layout.Horizontal,
			}.Layout(gtx,
				layout.Flexed(0.5, func(gtx C) D {
					return material.Label(theme, 20, "Toggle theme").Layout(gtx)
				}),
				layout.Flexed(0.5, func(gtx C) D {
					return material.Switch(theme, &ui.isLightMode, "Toggle theme").Layout(gtx)
				}),
			)
		}),
		// download path
		layout.Rigid(func(gtx C) D {
			return layout.Flex{
				Spacing: layout.SpaceBetween, Axis: layout.Horizontal,
			}.Layout(gtx,
				layout.Flexed(0.5, func(gtx C) D {
					return material.Label(theme, 20, "Download path").Layout(gtx)
				}),
				layout.Flexed(0.5, func(gtx C) D {
					return material.Button(theme, &ui.buttons[PICK_PATH], ui.downloadPath).Layout(gtx)
				}),
			)
		}),
		// show notifications
		layout.Rigid(func(gtx C) D {
			return material.CheckBox(theme, &ui.notifyOption, "Show notifications").Layout(gtx)
		}),
		// trust peers
		layout.Rigid(func(gtx C) D {
			return material.CheckBox(theme, &ui.trustOption, "Trust previous senders").Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			year := time.Now().Year()
			copyrightYear := "2025"
			if year != 2025 {
				copyrightYear = fmt.Sprintf("%s-%d", copyrightYear, year)
			}
			copyright := fmt.Sprintf("Abigail Adegbiji @aabiji, %s", copyrightYear)
			return material.Label(theme, 20, copyright).Layout(gtx)
		}),
	)
}
