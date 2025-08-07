package main

import (
	"image"
	"image/color"
	"os"

	"gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Styles struct {
	theme        *material.Theme
	rounding     int
	borderWidth  int
	fg400        color.NRGBA
	fg500        color.NRGBA
	bg400        color.NRGBA
	bg500        color.NRGBA
	border500    color.NRGBA
	border600    color.NRGBA
	red500       color.NRGBA
	green500     color.NRGBA
	primary400   color.NRGBA
	primary500   color.NRGBA
	primary600   color.NRGBA
	secondary500 color.NRGBA
	bgOverlay    color.NRGBA
	transparent  color.NRGBA
}

func loadFont(path string) ([]font.FontFace, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	faces, err := opentype.ParseCollection(contents)
	return faces, err
}

func NewStyles(darkMode bool) Styles {
	base := Styles{
		rounding: 12, borderWidth: 2, transparent: color.NRGBA{A: 0},
	}

	if darkMode {
		base.fg400 = color.NRGBA{R: 180, G: 180, B: 190, A: 255}
		base.fg500 = color.NRGBA{R: 245, G: 245, B: 245, A: 255}
		base.bg400 = color.NRGBA{R: 45, G: 45, B: 50, A: 255}
		base.bg500 = color.NRGBA{R: 28, G: 28, B: 30, A: 255}
		base.border500 = color.NRGBA{R: 50, G: 50, B: 60, A: 255}
		base.border600 = color.NRGBA{R: 90, G: 90, B: 100, A: 255}
		base.red500 = color.NRGBA{R: 255, G: 69, B: 58, A: 255}
		base.green500 = color.NRGBA{R: 0, G: 200, B: 0, A: 255}
		base.primary400 = color.NRGBA{R: 100, G: 170, B: 230, A: 255}
		base.primary500 = color.NRGBA{R: 10, G: 132, B: 255, A: 255}
		base.primary600 = color.NRGBA{R: 64, G: 156, B: 255, A: 255}
		base.secondary500 = color.NRGBA{R: 255, G: 105, B: 180, A: 255}
		base.bgOverlay = color.NRGBA{R: 0, G: 0, B: 0, A: 180}
	} else {
		base.fg400 = color.NRGBA{R: 80, G: 80, B: 90, A: 255}
		base.fg500 = color.NRGBA{R: 29, G: 29, B: 31, A: 255}
		base.bg400 = color.NRGBA{R: 235, G: 235, B: 240, A: 255}
		base.bg500 = color.NRGBA{R: 242, G: 242, B: 247, A: 255}
		base.border500 = color.NRGBA{R: 220, G: 220, B: 230, A: 255}
		base.border600 = color.NRGBA{R: 200, G: 200, B: 210, A: 255}
		base.red500 = color.NRGBA{R: 255, G: 59, B: 48, A: 255}
		base.green500 = color.NRGBA{R: 0, G: 200, B: 0, A: 255}
		base.primary400 = color.NRGBA{R: 90, G: 160, B: 225, A: 255}
		base.primary500 = color.NRGBA{R: 0, G: 122, B: 255, A: 255}
		base.primary600 = color.NRGBA{R: 10, G: 132, B: 255, A: 255}
		base.secondary500 = color.NRGBA{R: 255, G: 105, B: 180, A: 255}
		base.bgOverlay = color.NRGBA{R: 255, G: 255, B: 255, A: 180}
	}

	base.theme = material.NewTheme()
	base.theme.Fg = base.fg500

	roboto, err := loadFont("assets/roboto.ttf")
	if err != nil {
		panic(err)
	}
	base.theme.Shaper = text.NewShaper(text.WithCollection(roboto))
	return base
}

func XCentered(gtx C, fullWidth bool, w layout.Widget) D {
	return layout.Center.Layout(gtx, func(gtx C) D {
		if fullWidth {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
		}
		return w(gtx)
	})
}

func Text(gtx C, styles Styles, text string, size int, invert bool) D {
	style := material.Label(styles.theme, unit.Sp(size), text)
	style.Color = styles.fg500
	if invert {
		style.Color = styles.bg500
	}
	style.TextSize = unit.Sp(size)
	return style.Layout(gtx)
}

func ProgressBar(gtx C, styles Styles, progress float32) D {
	style := material.ProgressBar(styles.theme, progress)
	style.Color = styles.primary400
	style.TrackColor = styles.transparent
	style.Height = unit.Dp(6)
	return style.Layout(gtx)
}

func Spinner(gtx C, styles Styles, label string) D {
	textPadding := layout.Inset{Left: unit.Dp(18), Top: unit.Dp(4)}
	return layout.Flex{
		Alignment: layout.Middle,
		Axis:      layout.Horizontal,
		Spacing:   layout.SpaceEnd,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			size := image.Pt(gtx.Dp(unit.Dp(28)), gtx.Dp(unit.Dp(28)))
			gtx.Constraints.Min = size
			gtx.Constraints.Max = size
			loader := material.Loader(styles.theme)
			loader.Color = styles.primary500
			style := loader.Layout(gtx)
			style.Size = size
			return style
		}),
		layout.Rigid(func(gtx C) D {
			return textPadding.Layout(gtx, func(gtx C) D {
				return Text(gtx, styles, label, 20, false)
			})
		}),
	)
}

func Modal(gtx C, styles Styles, w layout.Widget) D {
	bgOverlay := func(gtx C, w layout.Widget) D {
		defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, styles.bgOverlay)
		return w(gtx)
	}
	return bgOverlay(gtx, func(gtx C) D {
		return layout.Center.Layout(gtx, func(gtx C) D {
			return Div{
				width:            85,
				height:           90,
				padding:          layout.UniformInset(unit.Dp(20)),
				background:       styles.bg500,
				borderRadius:     styles.rounding,
				centerHorizontal: true,
				centerVertical:   true,
			}.Layout(gtx, func(gtx C) D { return w(gtx) })
		})
	})
}

func Checkbox(
	gtx C, styles Styles, toggle *widget.Bool,
	icon *widget.Icon, label string) D {
	margin := layout.Inset{Bottom: unit.Dp(20)}
	textMargin := layout.Inset{Left: unit.Dp(14)}
	iconPadding := layout.Inset{
		Top: unit.Dp(1), Bottom: unit.Dp(1),
		Left: unit.Dp(1), Right: unit.Dp(1),
	}

	return margin.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Spacing:   layout.SpaceEnd,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				checkboxWidget := func(gtx C) D {
					size := image.Pt(gtx.Dp(unit.Dp(25)), gtx.Dp(unit.Dp(25)))
					gtx.Constraints.Min = size
					gtx.Constraints.Max = size

					circle := clip.Ellipse{Min: image.Pt(0, 0), Max: size}

					paint.FillShape(gtx.Ops, styles.border600,
						clip.Stroke{
							Path:  circle.Path(gtx.Ops),
							Width: float32(gtx.Dp(unit.Dp(1.5))),
						}.Op())

					if toggle.Hovered() || toggle.Update(gtx) {
						pointer.CursorPointer.Add(gtx.Ops)
					}
					if toggle.Value {
						paint.FillShape(gtx.Ops, styles.primary500,
							circle.Op(gtx.Ops))
						iconPadding.Layout(gtx, func(gtx C) D {
							return icon.Layout(gtx, styles.bg500)
						})
					}
					return D{Size: size}
				}

				return toggle.Layout(gtx, checkboxWidget)
			}),

			layout.Rigid(func(gtx C) D {
				return textMargin.Layout(gtx, func(gtx C) D {
					return Text(gtx, styles, label, 18, false)
				})
			}),
		)
	})
}

type Button struct {
	bgColor       color.NRGBA
	borderColor   color.NRGBA
	hoveredColor  color.NRGBA
	disabled      bool
	padding       layout.Inset
	margin        layout.Inset
	borderWidth   int
	roundedBorder bool
	clickable     *widget.Clickable
}

func (b Button) Layout(gtx C, w layout.Widget) D {
	return b.margin.Layout(gtx, func(gtx C) D {
		return b.clickable.Layout(gtx, func(gtx C) D {
			if (b.clickable.Hovered() || b.clickable.Clicked(gtx)) && !b.disabled {
				pointer.CursorPointer.Add(gtx.Ops)
			}
			dims := b.padding.Layout(gtx, w)

			radius := gtx.Dp(unit.Dp(0))
			if b.roundedBorder {
				radius = gtx.Dp(unit.Dp(8))
			}
			shape := clip.RRect{
				Rect: image.Rectangle{
					Max: image.Pt(dims.Size.X, dims.Size.Y),
				},
				NE: radius, NW: radius, SE: radius, SW: radius,
			}

			bg := b.bgColor
			if b.clickable.Hovered() {
				bg = b.hoveredColor
			}
			paint.FillShape(gtx.Ops, bg, shape.Op(gtx.Ops))

			if b.borderWidth > 0 {
				borderWidth := float32(gtx.Dp(unit.Dp(b.borderWidth)))
				paint.FillShape(gtx.Ops, b.borderColor, clip.Stroke{
					Path:  shape.Path(gtx.Ops),
					Width: borderWidth,
				}.Op())
			}

			return b.padding.Layout(gtx, func(gtx C) D { return w(gtx) })
		})
	})
}

func IconButton(
	gtx C, styles Styles, size int, inverted bool,
	clickable *widget.Clickable, icon *widget.Icon) D {
	return Button{
		bgColor:      styles.transparent,
		hoveredColor: styles.transparent,
		padding: layout.Inset{
			Top: unit.Dp(1), Bottom: unit.Dp(1),
			Left: unit.Dp(1), Right: unit.Dp(1),
		},
		borderWidth: 0,
		clickable:   clickable,
	}.Layout(gtx, func(gtx C) D {
		size := image.Pt(gtx.Dp(unit.Dp(size)), gtx.Dp(unit.Dp(size)))
		gtx.Constraints.Min = size
		gtx.Constraints.Max = size

		c := styles.fg400
		if inverted {
			c = styles.bg400
		}

		if clickable.Hovered() || clickable.Clicked(gtx) {
			c = styles.fg500
			if inverted {
				c = styles.bg500
			}
		}
		return icon.Layout(gtx, c)
	})
}

func TextButton(
	gtx C, styles Styles, text string, textSize int,
	inverted bool, disabled bool, rounded bool,
	clickable *widget.Clickable) D {
	bg := styles.primary500
	border := styles.border500
	hovered := styles.primary600
	width := 0

	if inverted {
		bg = styles.bg500
		border = styles.primary500
		width = 1
	} else if disabled {
		bg = styles.primary400
		hovered = styles.primary400
	}

	return Button{
		bgColor:       bg,
		borderColor:   border,
		hoveredColor:  hovered,
		roundedBorder: rounded,
		padding: layout.Inset{
			Top: unit.Dp(8), Bottom: unit.Dp(8),
			Left: unit.Dp(8), Right: unit.Dp(8),
		},
		borderWidth: width,
		disabled:    disabled,
		clickable:   clickable,
	}.Layout(gtx, func(gtx C) D {
		return layout.Center.Layout(gtx, func(gtx C) D {
			invert := (clickable.Hovered() && inverted) || !inverted
			return Text(gtx, styles, text, textSize, invert)
		})
	})
}

type Div struct {
	padding layout.Inset
	margin  layout.Inset

	width            int // as percentage
	height           int // as percentage
	centerHorizontal bool
	centerVertical   bool

	background   color.NRGBA
	borderColor  color.NRGBA
	borderRadius int
	borderWidth  int
}

func (d Div) Layout(gtx C, w layout.Widget) D {
	return d.margin.Layout(gtx, func(gtx C) D {
		// set size
		outer := gtx.Constraints.Max
		if d.width > 0 {
			w := d.width * outer.X / 100
			gtx.Constraints.Min.X = w
			gtx.Constraints.Max.X = w
		}
		if d.height > 0 {
			h := d.height * outer.Y / 100
			gtx.Constraints.Min.Y = h
			gtx.Constraints.Max.Y = h
		}

		// center
		var offsetX, offsetY int
		if d.centerHorizontal {
			offsetX = (outer.X - gtx.Constraints.Max.X) / 2
		}
		if d.centerVertical {
			offsetY = (outer.Y - gtx.Constraints.Max.Y) / 2
		}
		stack := op.Offset(image.Pt(offsetX, offsetY)).Push(gtx.Ops)
		defer stack.Pop()

		// measure content with padding
		contentDims := d.padding.Layout(gtx, w)

		r := gtx.Dp(unit.Dp(d.borderRadius))
		rounded := clip.RRect{
			Rect: image.Rectangle{Max: contentDims.Size},
			NE:   r, NW: r, SE: r, SW: r,
		}
		paint.FillShape(gtx.Ops, d.background, rounded.Op(gtx.Ops))

		if d.borderWidth > 0 {
			borderWidth := float32(gtx.Dp(unit.Dp(d.borderWidth)))
			paint.FillShape(gtx.Ops, d.borderColor, clip.Stroke{
				Path:  rounded.Path(gtx.Ops),
				Width: borderWidth,
			}.Op())
		}

		// redraw content to ensure it's on top
		return d.padding.Layout(gtx, w)
	})
}
