package main

import (
	"errors"
	"local/defaultapp/internal/association"
	"path/filepath"
	"testing"
	"time"
)

func TestFeedbackSortingAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	m := &model{service: association.New(nil), statePath: path, rows: []row{
		{kind: fileKind{"z", "Alpha", "编程"}, selected: true, typ: association.Type{ID: "test.z", Default: association.Application{ID: "old"}}},
		{kind: fileKind{"a", "Beta", "编程"}, typ: association.Type{ID: "test.a", Default: association.Application{ID: "old"}}},
		{kind: fileKind{"b", "Beta", "编程"}, typ: association.Type{ID: "test.b", Default: association.Application{ID: "old"}}},
	}}
	before := append([]row(nil), m.rows...)
	after := append([]row(nil), m.rows...)
	after[1].typ.Default.ID = "new"
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	m.recordResults(before, after, []association.Result{{Extension: "a", Message: "已修改"}, {Extension: "z", Err: errors.New("OS rejected")}}, false, at)
	if m.rows[1].outcome != 1 || m.rows[0].outcome != 2 {
		t.Fatal("missing success/failure feedback")
	}
	order := m.visibleRows()
	if m.rows[order[0]].kind.ext != "a" || m.rows[order[1]].kind.ext != "z" {
		t.Fatal("expected newest first, then names ascending")
	}
	if !m.rows[0].modified.IsZero() {
		t.Fatal("failed writes must not update modified time")
	}
	secondBefore := append([]row(nil), m.rows...)
	m.recordResults(secondBefore, append([]row(nil), m.rows...), []association.Result{{Extension: "b", Err: errors.New("second failure")}}, false, at.Add(time.Minute))
	if m.rows[1].outcome != 1 || m.rows[0].outcome != 2 {
		t.Fatal("feedback must remain until a tab switch")
	}
	m.switchCategory(1)
	if m.notice != "" || m.rows[0].result != "" || m.rows[1].outcome != 0 || !m.rows[0].selected || !m.rows[1].modified.Equal(at) {
		t.Fatal("tab switch must clear only transient feedback")
	}
	m.moveType("a", "其他")
	restored := newModel(association.New(nil), path)
	found := false
	for _, r := range restored.rows {
		if r.kind.ext == "a" {
			found = true
			if r.kind.category != "其他" || !r.modified.Equal(at) {
				t.Fatal("category/time did not survive restart")
			}
		}
	}
	if !found {
		t.Fatal("custom suffix did not survive restart")
	}
}
func TestNoOpDoesNotReorderType(t *testing.T) {
	m := &model{service: association.New(nil), rows: []row{{kind: fileKind{"txt", "Text", "办公"}, typ: association.Type{ID: "text", Default: association.Application{ID: "same"}}}}}
	before := append([]row(nil), m.rows...)
	after := append([]row(nil), m.rows...)
	m.recordResults(before, after, []association.Result{{Extension: "txt", Message: "已是目标应用，无需修改"}}, false, time.Now())
	if !m.rows[0].modified.IsZero() {
		t.Fatal("no-op must not claim a new modification")
	}
}
