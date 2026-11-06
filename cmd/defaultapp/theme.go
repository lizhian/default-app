package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
void defaultapp_theme_color(int index, float *rgba);
*/
import "C"

import "github.com/vivaui/viva/ui"

// Hallmark v2: redesign full; app screen; macOS semantic colours, system typography.
// Critique: hierarchy and restraint lead; compact rows replace oversized tiles.
// CSS-only checks are inapplicable: this is Viva's native canvas, not a web view.
func appTheme() ui.Theme {
	color := func(index int) ui.Color {
		var rgba [4]C.float
		C.defaultapp_theme_color(C.int(index), &rgba[0])
		return ui.Color{Red: float32(rgba[0]), Green: float32(rgba[1]), Blue: float32(rgba[2]), Alpha: float32(rgba[3])}
	}
	t := ui.LightTheme()
	t.Background = color(0)
	t.Foreground = color(1)
	t.Card = color(2)
	t.CardForeground = t.Foreground
	t.Muted = color(3)
	t.MutedForeground = color(4)
	t.Border = color(5)
	t.Primary = color(6)
	t.PrimaryForeground = color(7)
	t.Secondary = t.Card
	t.SecondaryForeground = t.Foreground
	t.Accent = color(8)
	t.AccentForeground = t.Foreground
	t.Selection = t.Accent
	t.SelectionForeground = t.Foreground
	t.Hover = t.Muted
	t.Input = t.MutedForeground
	t.Popover = t.Card
	t.PopoverForeground = t.Foreground
	t.CardElevated = t.Card
	t.Ring = t.Primary
	t.Success = color(9)
	t.Metrics.ControlHeight = 32
	t.Metrics.RadiusControl = 6
	t.Metrics.RadiusSurface = 8
	t.Metrics.FontSizeS = 11
	t.Metrics.FontSizeM = 13
	t.Metrics.FontSizeL = 20
	t.Transition = ui.Motion{}
	return t
}
