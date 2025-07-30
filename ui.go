package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

const (
	HOME_PAGE = iota
	SETTINGS_PAGE
	PROGRESS_PAGE
	PICKER_PAGE
)

const (
	BACK_ICON = iota
	GEAR_ICON
	UPLOAD_ICON
	CLOSE_ICON
	CHECK_ICON
)

const (
	BTNS_START = iota
	UPLOAD_BTN
	SEND_BTN
	THEME_BTN
	PATH_BTN
	PAGE_BTN
	BACK_BTN
	SELECT_BTN
	BTNS_END
)

type C = layout.Context
type D = layout.Dimensions
type T = *material.Theme

type FolderEntry struct {
	name      string
	clickable widget.Clickable
}

type FileEntry struct {
	name      string
	size      int64
	data      []byte
	progress  float32
	clickable widget.Clickable
}

type Peer struct {
	name  string
	check widget.Bool
}

type UI struct {
	picker *explorer.Explorer
	styles Styles

	peersList   *widget.List
	foldersList *widget.List
	filesList   *widget.List

	errors    []error
	files     []FileEntry
	folders   []FolderEntry
	peers     []Peer
	icons     []*widget.Icon
	buttons   []widget.Clickable
	errClicks []widget.Clickable

	trustPeers widget.Bool
	notifyUser widget.Bool
	lightMode  widget.Bool

	selectedPeers map[string]bool
	downloadPath  string
	currentPage   int
}

func NewUI(e *explorer.Explorer) *UI {
	ui := &UI{
		peersList: &widget.List{
			List: layout.List{Axis: layout.Vertical}},
		filesList: &widget.List{
			List: layout.List{Axis: layout.Vertical}},
		foldersList: &widget.List{
			List: layout.List{Axis: layout.Vertical}},
		buttons:       make([]widget.Clickable, BTNS_END-BTNS_START),
		styles:        NewStyles(false),
		selectedPeers: make(map[string]bool),
		currentPage:   HOME_PAGE,
		picker:        e,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	ui.downloadPath = filepath.Join(home, "Downloads")
	ui.setupFolderList()

	iconBytes := [][]byte{
		icons.NavigationArrowBack, icons.ActionSettings,
		icons.FileFileUpload, icons.NavigationClose,
		icons.NavigationCheck}
	for _, data := range iconBytes {
		icon, err := widget.NewIcon(data)
		if err != nil {
			panic(err)
		}
		ui.icons = append(ui.icons, icon)
	}
	return ui
}

func (ui *UI) AddError(err error) {
	ui.errors = append(ui.errors, err)
	ui.errClicks = append(ui.errClicks, widget.Clickable{})
}

func (ui *UI) addFiles() {
	selection, err := ui.picker.ChooseFiles()
	if err != nil {
		return
	}

	for _, readCloser := range selection {
		defer readCloser.Close()

		file, ok := readCloser.(fs.File)
		if !ok {
			panic("platform doesn't support file metadata")
		}

		info, err := file.Stat()
		if err != nil {
			panic(err)
		}

		entry := FileEntry{name: info.Name(), size: info.Size(), progress: -1}
		_, err = readCloser.Read(entry.data)
		if err != nil {
			continue
		}
		ui.files = append(ui.files, entry)
	}
}

func isWriteable(folderPath string) bool {
	temp := filepath.Join(folderPath, ".temp")
	file, err := os.Create(temp)
	if err != nil {
		return false
	}

	file.Close()
	return os.Remove(temp) == nil
}

func (ui *UI) setupFolderList() {
	entries, err := os.ReadDir(ui.downloadPath)
	if err != nil {
		panic(err)
	}

	ui.folders = nil
	for _, entry := range entries {
		fullpath := filepath.Join(ui.downloadPath, entry.Name())
		if entry.IsDir() && isWriteable(fullpath) {
			ui.folders = append(ui.folders, FolderEntry{name: entry.Name()})
		}
	}
}

func getCopyright() string {
	year := time.Now().Year()
	copyrightYear := "2025"
	if year != 2025 {
		copyrightYear = fmt.Sprintf("%s-%d", copyrightYear, year)
	}
	msg := "Made with ❤️ by Abigail Adegbiji @aabiji, %s"
	return fmt.Sprintf(msg, copyrightYear)
}

func (ui *UI) handleInputs(gtx C) {
	if ui.buttons[PAGE_BTN].Clicked(gtx) { // change the current page
		switch ui.currentPage {
		case HOME_PAGE:
			ui.currentPage = SETTINGS_PAGE
		case SETTINGS_PAGE, PROGRESS_PAGE:
			ui.currentPage = HOME_PAGE
		}
	}

	for i := len(ui.errors) - 1; i >= 0; i-- { // handle removing errors
		if ui.errClicks[i].Clicked(gtx) {
			ui.errors = append(ui.errors[:i], ui.errors[i+1:]...)
			ui.errClicks = append(ui.errClicks[:i], ui.errClicks[i+1:]...)
		}
	}

	for i := len(ui.files) - 1; i >= 0; i-- { // handle removing files
		if ui.files[i].clickable.Clicked(gtx) {
			ui.files = append(ui.files[:i], ui.files[i+1:]...)
		}
	}

	for i := len(ui.peers) - 1; i >= 0; i-- { // handle peer selection
		if ui.peers[i].check.Value {
			ui.selectedPeers[ui.peers[i].name] = true
		} else {
			delete(ui.selectedPeers, ui.peers[i].name)
		}
	}

	for i := 0; i < len(ui.folders); i++ { // navigate folders
		if ui.folders[i].clickable.Clicked(gtx) {
			ui.downloadPath = filepath.Join(ui.downloadPath, ui.folders[i].name)
			ui.setupFolderList()
			break
		}
	}

	if ui.buttons[BACK_BTN].Clicked(gtx) {
		ui.downloadPath = filepath.Dir(ui.downloadPath)
		ui.setupFolderList() // navigate one folder back up
	} else if ui.buttons[SELECT_BTN].Clicked(gtx) {
		ui.currentPage = SETTINGS_PAGE // close the folder picker
	} else if ui.buttons[PATH_BTN].Clicked(gtx) {
		ui.currentPage = PICKER_PAGE // show the folder picker
	}

	if ui.buttons[UPLOAD_BTN].Clicked(gtx) {
		go func() { ui.addFiles() }()
	}

	if ui.lightMode.Update(gtx) { // toggle theme
		ui.styles = NewStyles(ui.lightMode.Value)
	}
}

func (ui *UI) DrawFrame(gtx C) {
	ui.handleInputs(gtx)

	// draw the background
	paint.Fill(gtx.Ops, ui.styles.bg500)

	layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D { // navigation icon
			return layout.Inset{
				Top: 16, Left: 16,
			}.Layout(gtx, func(gtx C) D {
				i := ui.icons[GEAR_ICON]
				if ui.currentPage != HOME_PAGE {
					i = ui.icons[BACK_ICON]
				}
				return IconButton(gtx, ui.styles, 32, false,
					&ui.buttons[PAGE_BTN], i)
			})
		}),

		layout.Stacked(func(gtx C) D { // page content
			return Div{
				width:            75,
				height:           90,
				centerHorizontal: true,
				centerVertical:   true,
			}.Layout(gtx, func(gtx C) D {
				switch ui.currentPage {
				case HOME_PAGE:
					return ui.drawHomePage(gtx)
				case PROGRESS_PAGE:
					return ui.drawProgressPage(gtx)
				default:
					return ui.drawSettingsPage(gtx)
				}
			})
		}),

		layout.Stacked(func(gtx C) D { // error tray
			return layout.Flex{
				Axis:      layout.Vertical,
				Alignment: layout.Middle,
			}.Layout(gtx,
				// pushes error tray to the bottom
				layout.Flexed(1, func(gtx C) D {
					return layout.Spacer{}.Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return Div{
						width:            30,
						height:           30,
						background:       ui.styles.transparent,
						centerHorizontal: true,
					}.Layout(gtx, func(gtx C) D {
						return ui.drawErrorContainer(gtx)
					})
				}),
			)
		}),

		layout.Stacked(func(gtx C) D { // folder picker modal
			if ui.currentPage != PICKER_PAGE {
				return layout.Dimensions{}
			}
			return ui.drawFolderPicker(gtx)
		}),
	)
}

func (ui *UI) drawErrorContainer(gtx C) D {
	innerPadding := layout.Inset{
		Top: unit.Dp(8), Bottom: unit.Dp(8),
		Left: unit.Dp(15), Right: unit.Dp(10),
	}
	widgets := []layout.FlexChild{}

	// toast notifications
	for i := len(ui.errors) - 1; i >= 0; i-- {
		widgets = append(widgets,
			layout.Rigid(func(gtx C) D {
				return Div{
					padding:      innerPadding,
					margin:       layout.Inset{Bottom: unit.Dp(25)},
					background:   ui.styles.red500,
					borderRadius: ui.styles.rounding,
					width:        100,
					height:       35,
				}.Layout(gtx, func(gtx C) D {
					return layout.Center.Layout(gtx,
						func(gtx layout.Context) D {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return layout.Flex{
								Axis:      layout.Horizontal,
								Spacing:   layout.SpaceBetween,
								Alignment: layout.Middle,
							}.Layout(gtx,
								layout.Rigid(func(gtx C) D {
									return Text(gtx, ui.styles,
										ui.errors[i].Error(), 18, true)
								}),
								layout.Rigid(func(gtx C) D {
									return IconButton(gtx, ui.styles,
										20, true, &ui.errClicks[i],
										ui.icons[CLOSE_ICON])
								}),
							)
						})
				})
			}),
		)
	}

	return layout.Flex{
		Axis:      layout.Vertical,
		Spacing:   layout.SpaceStart,
		Alignment: layout.Middle,
	}.Layout(gtx, widgets...)
}

func (ui *UI) drawFileEntry(gtx C, file *FileEntry) D {
	padding := layout.Inset{Top: unit.Dp(16), Right: unit.Dp(16)}
	if file.progress <= 0 {
		// since there wouldn't be a progress bar at the bottom
		padding.Bottom = unit.Dp(16)
	}

	return Div{
		padding:      padding,
		margin:       layout.Inset{Bottom: unit.Dp(10)},
		width:        100,
		background:   ui.styles.bg500,
		borderColor:  ui.styles.border500,
		borderRadius: 10,
		borderWidth:  ui.styles.borderWidth,
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Flex{
					Axis:      layout.Horizontal,
					Spacing:   layout.SpaceBetween,
					Alignment: layout.Middle,
				}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Inset{Left: unit.Dp(16)}.Layout(gtx,
							func(gtx C) D {
								return Text(gtx, ui.styles, file.name,
									18, false)
							})
					}),
					layout.Rigid(func(gtx C) D {
						if file.progress < 0 {
							return IconButton(gtx, ui.styles, 20, false,
								&file.clickable, ui.icons[CLOSE_ICON])
						}
						// empty space so layout doesn't collapse
						return layout.Spacer{Width: unit.Dp(0)}.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(func(gtx C) D {
				if file.progress >= 0 {
					return layout.Inset{
						Top: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
						return ProgressBar(gtx, ui.styles, file.progress)
					})
				}
				// empty space so layout doesn't collapse
				return layout.Spacer{Height: unit.Dp(0)}.Layout(gtx)
			}),
		)
	})
}

func (ui *UI) drawUploadButton(gtx C) D {
	return Button{
		bgColor:      ui.styles.bg500,
		borderColor:  ui.styles.border500,
		hoveredColor: ui.styles.bg400,
		margin: layout.Inset{
			Top: unit.Dp(12), Bottom: unit.Dp(12),
		},
		padding:       layout.UniformInset(unit.Dp(15)),
		borderWidth:   ui.styles.borderWidth,
		roundedBorder: true,
		clickable:     &ui.buttons[UPLOAD_BTN],
	}.Layout(gtx, func(gtx C) D {
		return layout.Center.Layout(gtx, func(gtx C) D {
			return layout.Flex{
				Alignment: layout.Middle,
				Axis:      layout.Vertical,
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					icon := ui.icons[UPLOAD_ICON]
					gtx.Constraints.Min.X = 125
					gtx.Constraints.Min.Y = 125
					return icon.Layout(gtx, ui.styles.fg400)
				}),
				layout.Rigid(func(gtx C) D {
					return Text(gtx, ui.styles,
						"Select files", 20, false)
				}),
			)
		})
	})
}

func (ui *UI) drawPeersList() layout.FlexChild {
	widget := func(gtx C) D {
		if len(ui.peers) == 0 {
			return Spinner(gtx, ui.styles, "Looking for peers...")
		}

		return material.List(ui.styles.theme, ui.peersList).Layout(gtx, len(ui.peers),
			func(gtx C, i int) D {
				return Checkbox(gtx, ui.styles, &ui.peers[i].check,
					ui.icons[CHECK_ICON], ui.peers[i].name)
			})
	}

	if len(ui.peers) > 0 {
		return layout.Flexed(0.5, widget)
	}
	return layout.Rigid(widget)
}

func (ui *UI) drawHomePage(gtx C) D {
	widgets := []layout.FlexChild{
		ui.drawPeersList(),
		layout.Rigid(func(gtx C) D { return ui.drawUploadButton(gtx) }),
	}

	if len(ui.files) > 0 {
		widgets = append(widgets,
			layout.Flexed(0.5, func(gtx C) D { // list of files
				return material.List(ui.styles.theme, ui.filesList).Layout(gtx,
					len(ui.files), func(gtx C, i int) D {
						return ui.drawFileEntry(gtx, &ui.files[i])
					})
			}))
	}

	widgets = append(widgets,
		layout.Rigid(func(gtx C) D {
			disabled := len(ui.files) == 0 || len(ui.selectedPeers) == 0
			return TextButton(gtx, ui.styles, "Send files", 18, false, disabled,
				true, &ui.buttons[SEND_BTN])
		}),
	)

	return layout.Flex{
		Axis:      layout.Vertical,
		Spacing:   layout.SpaceEnd,
		Alignment: layout.Middle,
	}.Layout(gtx, widgets...)
}

func (ui *UI) drawProgressPage(gtx C) D {
	return layout.Flex{
		Alignment: layout.Start,
		Axis:      layout.Vertical,
		Spacing:   layout.SpaceEvenly,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Inset{Bottom: unit.Dp(20)}.Layout(gtx,
				func(gtx C) D {
					return layout.Center.Layout(gtx, func(gtx C) D {
						return Text(gtx, ui.styles,
							"Sending files", 40, false)
					})
				})
		}),

		layout.Flexed(0.9, func(gtx C) D {
			return material.List(ui.styles.theme, ui.filesList).Layout(gtx,
				len(ui.files), func(gtx C, i int) D {
					return ui.drawFileEntry(gtx, &ui.files[i])
				})
		}),
	)
}

func (ui *UI) drawSettingsPage(gtx C) D {
	return layout.Flex{
		Axis:      layout.Vertical,
		Spacing:   layout.SpaceEnd,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D { // toggle theme
			return Checkbox(gtx, ui.styles, &ui.lightMode,
				ui.icons[CHECK_ICON], "Dark mode")
		}),
		layout.Rigid(func(gtx C) D { // show notifications
			return Checkbox(gtx, ui.styles, &ui.notifyUser,
				ui.icons[CHECK_ICON], "Show notifications")
		}),
		layout.Rigid(func(gtx C) D { // trust peers
			return Checkbox(gtx, ui.styles, &ui.trustPeers,
				ui.icons[CHECK_ICON], "Trust previous senders")
		}),
		layout.Rigid(func(gtx C) D { // choose download path
			return layout.Flex{
				Spacing: layout.SpaceBetween, Axis: layout.Horizontal,
			}.Layout(gtx,
				layout.Flexed(0.5, func(gtx C) D {
					return Text(gtx, ui.styles, "Download path", 20, false)
				}),
				layout.Flexed(0.5, func(gtx C) D {
					return TextButton(gtx, ui.styles, ui.downloadPath,
						16, true, false, true, &ui.buttons[PATH_BTN])
				}),
			)
		}),
		layout.Rigid(func(gtx C) D { // copyright
			return layout.Inset{Top: unit.Dp(20)}.Layout(gtx, func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					return Text(gtx, ui.styles, getCopyright(), 14, false)
				})
			})
		}),
	)
}

func (ui *UI) drawFolderPicker(gtx C) D {
	return Modal(gtx, ui.styles, func(gtx C) D {
		return layout.Flex{
			Axis:      layout.Vertical,
			Spacing:   layout.SpaceEnd,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Flexed(0.5, func(gtx C) D {
				return material.List(ui.styles.theme, ui.foldersList).Layout(gtx,
					len(ui.folders), func(gtx C, i int) D {
						return TextButton(gtx, ui.styles, ui.folders[i].name,
							14, true, false, false, &ui.folders[i].clickable)
					})
			}),

			layout.Rigid(func(gtx C) D {
				return layout.Flex{
					Axis:      layout.Horizontal,
					Spacing:   layout.SpaceBetween,
					Alignment: layout.Middle,
				}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						proportional := 20 - (0.1 * float32(len(ui.downloadPath)))
						size := int(max(12, proportional))
						return Text(gtx, ui.styles, ui.downloadPath, size, false)
					}),

					layout.Rigid(func(gtx C) D {
						return layout.Flex{
							Axis:    layout.Horizontal,
							Spacing: layout.SpaceBetween,
						}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return TextButton(gtx, ui.styles, "Back", 15,
									false, false, true, &ui.buttons[BACK_BTN])
							}),
							layout.Rigid(func(gtx C) D {
								return layout.Spacer{Width: unit.Dp(20)}.Layout(gtx)
							}),
							layout.Rigid(func(gtx C) D {
								return TextButton(gtx, ui.styles, "Select", 15,
									false, false, true, &ui.buttons[SELECT_BTN])
							}),
						)
					}),
				)
			}),
		)
	})
}
