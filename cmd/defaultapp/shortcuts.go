package main

import (
	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"
	"local/defaultapp/internal/association"
	"sort"
	"strings"
)

func (m *model) editCategory(ext string, anchor ui.Rect) {
	if m.busy {
		return
	}
	var items []association.Application
	current := ""
	for _, r := range m.rows {
		if r.kind.ext == ext {
			current = r.kind.category
			break
		}
	}
	for _, category := range categories[1:] {
		items = append(items, association.Application{ID: category, Name: category})
	}
	picked, err := pickMenu(items, current, anchor)
	if err != nil {
		m.notice = err.Error()
		return
	}
	if picked >= 0 && picked < len(items) {
		m.moveType(ext, items[picked].ID)
	}
}
func (m *model) quickApplication(ext string, anchor ui.Rect) {
	if m.busy {
		return
	}
	current := association.Application{}
	for _, r := range m.rows {
		if r.kind.ext == ext {
			current = r.typ.Default
			break
		}
	}
	m.run(func() func() {
		apps, err := m.service.Applications(ext)
		if current.ID != "" && current.Path != "" {
			found := false
			for _, a := range apps {
				if strings.EqualFold(a.ID, current.ID) {
					found = true
				}
			}
			if !found {
				apps = append(apps, current)
			}
		}
		sort.Slice(apps, func(i, j int) bool { return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name) })
		for _, a := range apps {
			m.cacheIcon(a.Path)
		}
		return func() {
			if err != nil {
				m.notice = err.Error()
				return
			}
			m.window.Post(func() {
				items := append(append([]association.Application(nil), apps...), association.Application{ID: "choose-other", Name: "选择其他应用…"})
				picked, err := pickMenu(items, current.ID, anchor)
				if err != nil {
					m.notice = err.Error()
					return
				}
				if picked < 0 {
					return
				}
				if picked < len(apps) {
					m.applyChange([]string{ext}, apps[picked], false)
					return
				}
				paths, err := app.OpenFiles(app.OpenFileOptions{Title: "选择 ." + ext + " 的默认应用", Filters: []app.FileFilter{{Name: "macOS 应用", Pattern: "*.app"}}})
				if err != nil {
					m.notice = err.Error()
					return
				}
				if len(paths) > 0 {
					m.quickApplicationPath(ext, paths[0])
				}
			})
		}
	})
}
func (m *model) quickApplicationPath(ext, path string) {
	m.run(func() func() {
		target, err := association.ApplicationAtPath(path)
		if err == nil {
			err = association.RegisterApplication(path)
			m.cacheIcon(path)
		}
		return func() {
			if err != nil {
				m.notice = err.Error()
				return
			}
			m.window.Post(func() { m.applyChange([]string{ext}, target, false) })
		}
	})
}
