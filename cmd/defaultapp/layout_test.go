package main

import (
	"github.com/vivaui/viva/ui"
	"testing"
)

func TestLoadingKeepsTableBounds(t *testing.T) {
	m := &model{}
	canvas := &ui.Canvas{}
	canvas.SetTree(&tree)
	canvas.SetMeasurer(func(text string, size, weight float32) float32 { return float32(len(text)) * size / 2 })
	config := m.config()
	config.Paint(canvas, ui.Size{Width: 980, Height: 700})
	idle := m.table.Rect()
	m.busy = true
	config.Paint(canvas, ui.Size{Width: 980, Height: 700})
	if got := m.table.Rect(); got != idle {
		t.Fatalf("loading shifts table: idle=%+v loading=%+v", idle, got)
	}
}
