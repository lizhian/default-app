package main

import (
	"math"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/flex"
	"github.com/vivaui/viva/ui"
)

// These calls touch the system icon database, so only the worker calls cacheIcon.
func (m *model) cacheIcon(path string) {
	if path == "" {
		return
	}
	if _, ok := m.icons.Load(path); ok {
		return
	}
	bitmap, err := app.FileIcon(path, 40)
	if err != nil {
		return
	}
	bitmap.ID = m.nextIcon.Add(1)
	m.icons.Store(path, bitmap)
}
func (m *model) appIcon(f *flex.Frame, path string) {
	if value, ok := m.icons.Load(path); ok {
		f.Image(value.(ui.Bitmap)).Size(20, 20)
	} else {
		f.Slot().Size(20, 20)
	}
}

var outsideCorner = ui.Icon{ViewBox: 1, Commands: []ui.PathCommand{
	ui.MoveTo(0, 0), ui.LineTo(1, 0), ui.CubicTo(0.44771525, 0, 0, 0.44771525, 0, 1), ui.ClosePath(),
}}

// Viva's clip is rectangular. Cover only the outside of each rounded corner
// after the children, then paint the outline once above their square backgrounds.
func paintTableCorners(canvas *ui.Canvas, r ui.Rect, theme ui.Theme) {
	const radius float32 = 8
	corners := [4]ui.Rect{
		{X: r.X, Y: r.Y, Width: radius, Height: radius},
		{X: r.X + r.Width - radius, Y: r.Y, Width: radius, Height: radius},
		{X: r.X + r.Width - radius, Y: r.Y + r.Height - radius, Width: radius, Height: radius},
		{X: r.X, Y: r.Y + r.Height - radius, Width: radius, Height: radius},
	}
	canvas.PushLayer(1)
	for i, corner := range corners {
		canvas.DrawIconTransformed(outsideCorner, corner, theme.Background, ui.IconTransform{Rotate: float32(i) * math.Pi / 2})
	}
	b := canvas.PathBuilder()
	w, h := r.Width, r.Height
	const half float32 = 0.5
	const k float32 = 3.85786
	b.MoveTo(radius, half)
	b.LineTo(w-radius, half)
	b.CubicTo(w-k, half, w-half, k, w-half, radius)
	b.LineTo(w-half, h-radius)
	b.CubicTo(w-half, h-k, w-k, h-half, w-radius, h-half)
	b.LineTo(radius, h-half)
	b.CubicTo(k, h-half, half, h-k, half, h-radius)
	b.LineTo(half, radius)
	b.CubicTo(half, k, k, half, radius, half)
	b.Close()
	canvas.DrawPath(b.Path(w, h), r, ui.PathStyle{Stroke: theme.Border, StrokeWidth: 1})
	canvas.PopLayer()
}
