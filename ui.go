package main

import (
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
	"github.com/aabiji/drip/p2p"
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
	ACCEPT_BTN
	DENY_BTN
	BTNS_END
)

type C = layout.Context
type D = layout.Dimensions

type Item struct {
	name      string
	clickable widget.Clickable
	check     widget.Bool

	// file info
	rc       io.ReadCloser
	size     int64
	progress float32
}

type UI struct {
	settings  *Settings
	isAndroid bool
	appEvents chan p2p.Message
	picker    *explorer.Explorer
	styles    Styles

	recipientsList *widget.List
	recipients     []Item
	foldersList    *widget.List
	folders        []Item
	filesList      *widget.List
	files          []Item

	errors  []Item
	icons   []*widget.Icon
	buttons []widget.Clickable

	currentPage   int
	authMsg       string
	showAuthPopup bool
	sendingMsg    string
	sendingDone   bool
}

func NewUI(s *Settings, appEvents chan p2p.Message, isAndroid bool) *UI {
	ui := &UI{
		appEvents: appEvents,
		recipientsList: &widget.List{
			List: layout.List{Axis: layout.Vertical}},
		filesList: &widget.List{
			List: layout.List{Axis: layout.Vertical}},
		foldersList: &widget.List{
			List: layout.List{Axis: layout.Vertical}},
		buttons:     make([]widget.Clickable, BTNS_END-BTNS_START),
		currentPage: HOME_PAGE,
		settings:    s,
		styles:      NewStyles(s.DarkMode.Value),
		isAndroid:   isAndroid,
	}

	if !isAndroid {
		ui.setupFolderList()
	}

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

func (ui *UI) UpdateRecipients(recipient string, remove bool) {
	// TODO: get the UI to immediately update
	if !remove {
		ui.recipients = append(ui.recipients, Item{name: recipient})
	} else {
		for i := 0; i < len(ui.recipients); i++ {
			if ui.recipients[i].name == recipient {
				ui.recipients = append(ui.recipients[:i], ui.recipients[i+1:]...)
				break
			}
		}
	}
}

func (ui *UI) UpdateFileProgresses(percentages map[string]float32) {
	for i := 0; i < len(ui.files); i++ {
		name := ui.files[i].name
		ui.files[i].progress = percentages[name]
	}
}

func (ui *UI) AddError(err string) { ui.errors = append(ui.errors, Item{name: err}) }

func (ui *UI) ForgetCurrentTransfer(cancel bool, empty bool) {
	ui.currentPage = HOME_PAGE
	if cancel {
		ui.appEvents <- p2p.NewMessage(p2p.TRANSFER_CANCELLED, "")
	}
	if !empty {
		return
	}
	for i := 0; i < len(ui.recipients); i++ {
		ui.recipients[i].check.Value = false
	}
	ui.files = []Item{}
}

func (ui *UI) sendBtnDisabled() bool {
	noPeersSelected := true
	for _, peer := range ui.recipients {
		if peer.check.Value {
			noPeersSelected = false
			break
		}
	}
	return len(ui.files) == 0 || noPeersSelected
}

func (ui *UI) addFiles() {
	selection, err := ui.picker.ChooseFiles()
	if err != nil {
		return
	}

	for _, readCloser := range selection {
		file, ok := readCloser.(fs.File)
		if !ok {
			panic("platform doesn't support file metadata")
		}

		info, err := file.Stat()
		if err != nil {
			panic(err)
		}

		n := fmt.Sprintf(
			"%s-%d%s", strings.Split(info.Name(), ".")[0],
			rand.IntN(100), filepath.Ext(info.Name()))
		ui.files = append(ui.files, Item{
			name: n, size: info.Size(), rc: readCloser, progress: -1})
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
	entries, err := os.ReadDir(ui.settings.DownloadPath)
	if err != nil {
		panic(err)
	}

	ui.folders = nil
	for _, entry := range entries {
		fullpath := filepath.Join(ui.settings.DownloadPath, entry.Name())
		if entry.IsDir() && isWriteable(fullpath) {
			ui.folders = append(ui.folders, Item{name: entry.Name()})
		}
	}
}

func getCopyright() string {
	year := time.Now().Year()
	copyrightYear := "2025"
	if year != 2025 {
		copyrightYear = fmt.Sprintf("%s-%d", copyrightYear, year)
	}
	return fmt.Sprintf("Abigail Adegbiji, %s", copyrightYear)
}

func (ui *UI) handleInputs(gtx C) {
	if ui.buttons[PAGE_BTN].Clicked(gtx) { // change the current page
		if ui.currentPage == HOME_PAGE || ui.currentPage == SETTINGS_PAGE {
			ui.currentPage = (ui.currentPage + 1) % 2
		} else if ui.currentPage == PROGRESS_PAGE {
			ui.ForgetCurrentTransfer(!ui.sendingDone, true)
		}
	}

	for i := len(ui.errors) - 1; i >= 0; i-- { // handle removing errors
		if ui.errors[i].clickable.Clicked(gtx) {
			ui.errors = append(ui.errors[:i], ui.errors[i+1:]...)
		}
	}

	for i := len(ui.files) - 1; i >= 0; i-- { // handle removing files
		if ui.files[i].clickable.Clicked(gtx) {
			ui.files = append(ui.files[:i], ui.files[i+1:]...)
		}
	}

	if !ui.sendBtnDisabled() && ui.buttons[SEND_BTN].Clicked(gtx) {
		ui.appEvents <- p2p.NewMessage(p2p.SEND_FILES, "")
	}

	acceptClicked := ui.buttons[ACCEPT_BTN].Clicked(gtx)
	if acceptClicked || ui.buttons[DENY_BTN].Clicked(gtx) {
		ui.appEvents <- p2p.NewMessage(p2p.AUTH_GRANTED, acceptClicked)
	}

	if ui.buttons[UPLOAD_BTN].Clicked(gtx) {
		go func() { ui.addFiles() }()
	}

	if ui.settings.DarkMode.Update(gtx) { // toggle theme
		ui.styles = NewStyles(ui.settings.DarkMode.Value)
	}

	// handle the folder picker ui
	if !ui.isAndroid {
		for i := 0; i < len(ui.folders); i++ { // navigate folders
			if ui.folders[i].clickable.Clicked(gtx) {
				ui.settings.DownloadPath =
					filepath.Join(ui.settings.DownloadPath, ui.folders[i].name)
				ui.setupFolderList()
				break
			}
		}
		if ui.buttons[BACK_BTN].Clicked(gtx) {
			ui.settings.DownloadPath = filepath.Dir(ui.settings.DownloadPath)
			ui.setupFolderList() // navigate one folder back up
		} else if ui.buttons[SELECT_BTN].Clicked(gtx) {
			ui.currentPage = SETTINGS_PAGE // close the folder picker
		} else if ui.buttons[PATH_BTN].Clicked(gtx) {
			ui.currentPage = PICKER_PAGE // show the folder picker
		}
	}
}

func (ui *UI) DrawFrame(gtx C) {
	ui.handleInputs(gtx)

	// draw the background
	paint.Fill(gtx.Ops, ui.styles.bg500)

	layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D { // navigation icon
			if ui.showAuthPopup {
				return layout.Dimensions{}
			}

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
			if ui.showAuthPopup {
				return layout.Dimensions{}
			}

			widthPercent := 75
			if gtx.Constraints.Max.X <= 300 {
				widthPercent = 96
			}

			return Div{
				padding:          layout.Inset{Top: unit.Dp(35)},
				width:            widthPercent,
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
			if ui.showAuthPopup {
				return layout.Dimensions{}
			}

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
			if ui.currentPage != PICKER_PAGE || ui.isAndroid {
				return layout.Dimensions{}
			}
			return ui.drawFolderPicker(gtx)
		}),

		layout.Stacked(func(gtx C) D { // auth transfer modal
			if ui.showAuthPopup {
				return ui.drawPermissionPage(gtx)
			} else {
				return layout.Dimensions{}
			}
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
					return XCentered(gtx, false,
						func(gtx layout.Context) D {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return layout.Flex{
								Axis:      layout.Horizontal,
								Spacing:   layout.SpaceBetween,
								Alignment: layout.Middle,
							}.Layout(gtx,
								layout.Rigid(func(gtx C) D {
									return Text(gtx, ui.styles,
										ui.errors[i].name, 18, true)
								}),
								layout.Rigid(func(gtx C) D {
									return IconButton(gtx, ui.styles,
										20, true, &ui.errors[i].clickable,
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

func (ui *UI) drawFileEntry(gtx C, file *Item) D {
	padding := layout.Inset{Top: unit.Dp(16), Right: unit.Dp(16)}
	if file.progress <= 0 {
		padding.Bottom = unit.Dp(16) // since there wouldn't be a progress bar
	} else {
		padding.Right = unit.Dp(0) // since there wouldn't be a close button
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
				if file.progress > 0 {
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
		return XCentered(gtx, false, func(gtx C) D {
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

func (ui *UI) drawHomePage(gtx C) D {
	widgets := []layout.FlexChild{
		layout.Rigid(func(gtx C) D {
			if len(ui.recipients) == 0 {
				return Spinner(gtx, ui.styles, "Looking for recipients...")
			}
			list := material.List(ui.styles.theme, ui.recipientsList)
			list.AnchorStrategy = material.Overlay
			return list.Layout(gtx,
				len(ui.recipients), func(gtx C, i int) D {
					gtx.Constraints.Max.Y = gtx.Constraints.Max.Y * 50 / 100
					return Checkbox(gtx, ui.styles, &ui.recipients[i].check,
						ui.icons[CHECK_ICON], ui.recipients[i].name)
				})
		}),
		layout.Rigid(func(gtx C) D { return ui.drawUploadButton(gtx) }),
	}

	if len(ui.files) > 0 {
		widgets = append(widgets,
			layout.Rigid(func(gtx C) D { // list of files
				gtx.Constraints.Max.Y = gtx.Constraints.Max.Y * 50 / 100
				list := material.List(ui.styles.theme, ui.filesList)
				list.AnchorStrategy = material.Overlay
				return list.Layout(gtx,
					len(ui.files), func(gtx C, i int) D {
						return ui.drawFileEntry(gtx, &ui.files[i])
					})
			}))
	}

	widgets = append(widgets,
		layout.Rigid(func(gtx C) D {
			return TextButton(gtx, ui.styles, "Send files", 18, false,
				ui.sendBtnDisabled(), true, &ui.buttons[SEND_BTN])
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
					return XCentered(gtx, false, func(gtx C) D {
						return Text(gtx, ui.styles, ui.sendingMsg, 40, false)
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
			return Checkbox(gtx, ui.styles, &ui.settings.DarkMode,
				ui.icons[CHECK_ICON], "Dark mode")
		}),
		layout.Rigid(func(gtx C) D { // show notifications
			return Checkbox(gtx, ui.styles, &ui.settings.NotifyUser,
				ui.icons[CHECK_ICON], "Show notifications")
		}),
		layout.Rigid(func(gtx C) D { // trust recipients
			return Checkbox(gtx, ui.styles, &ui.settings.TrustPeers,
				ui.icons[CHECK_ICON], "Trust previous senders")
		}),
		layout.Rigid(func(gtx C) D { // choose download path
			// folder selection will be a desktop only feature
			// because i can't figure out how to open android's
			// folder picker through the jni bride
			if ui.isAndroid {
				return layout.Dimensions{}
			}

			return layout.Flex{
				Spacing: layout.SpaceBetween, Axis: layout.Horizontal,
			}.Layout(gtx,
				layout.Flexed(0.5, func(gtx C) D {
					return Text(gtx, ui.styles, "Download path", 20, false)
				}),
				layout.Flexed(0.5, func(gtx C) D {
					return TextButton(gtx, ui.styles, ui.settings.DownloadPath,
						16, true, false, true, &ui.buttons[PATH_BTN])
				}),
			)
		}),
		layout.Rigid(func(gtx C) D { // copyright
			return layout.Inset{Top: unit.Dp(20)}.Layout(gtx, func(gtx C) D {
				return XCentered(gtx, false, func(gtx C) D {
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
						proportional :=
							20 - (0.1 * float32(len(ui.settings.DownloadPath)))
						size := int(max(12, proportional))
						return Text(gtx, ui.styles, ui.settings.DownloadPath, size, false)
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

func (ui *UI) drawPermissionPage(gtx C) D {
	return Modal(gtx, ui.styles, func(gtx C) D {
		return XCentered(gtx, true, func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Vertical,
				Spacing:   layout.SpaceStart,
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return XCentered(gtx, false, func(gtx C) D {
						return Text(gtx, ui.styles, ui.authMsg, 30, false)
					})
				}),

				layout.Rigid(func(gtx C) D {
					return layout.Spacer{Height: unit.Dp(30)}.Layout(gtx)
				}),

				layout.Rigid(func(gtx C) D {
					return TextButton(gtx, ui.styles, "Yes", 18,
						false, false, true, &ui.buttons[ACCEPT_BTN])
				}),

				layout.Rigid(func(gtx C) D {
					return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
				}),

				layout.Rigid(func(gtx C) D {
					return TextButton(gtx, ui.styles, "No", 18,
						true, false, true, &ui.buttons[DENY_BTN])
				}),
			)
		})
	})
}
