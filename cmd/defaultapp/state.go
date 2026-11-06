package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"local/defaultapp/internal/association"
)

type savedType struct {
	Category string    `json:"category"`
	Name     string    `json:"name,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
}
type preferences struct {
	Types map[string]savedType `json:"types"`
}

func defaultStatePath() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "dev.local.defaultapp", "state.json")
}
func newModel(service *association.Service, path string) *model {
	m := &model{service: service, statePath: path}
	for _, kind := range catalog {
		m.rows = append(m.rows, row{kind: kind})
	}
	if path == "" {
		return m
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return m
	}
	if err != nil {
		m.notice = "读取本机配置失败：" + err.Error()
		return m
	}
	var p preferences
	if err = json.Unmarshal(data, &p); err != nil {
		m.notice = "读取本机配置失败：" + err.Error()
		return m
	}
	for ext, saved := range p.Types {
		normalized, err := association.Normalize(ext)
		if err != nil || ext != normalized || !validCategory(saved.Category) {
			continue
		}
		found := false
		for i := range m.rows {
			if m.rows[i].kind.ext == ext {
				m.rows[i].kind.category = saved.Category
				m.rows[i].modified = saved.Modified
				found = true
				break
			}
		}
		if !found && saved.Name != "" {
			m.rows = append(m.rows, row{kind: fileKind{ext, saved.Name, saved.Category}, modified: saved.Modified})
		}
	}
	return m
}
func validCategory(category string) bool {
	for _, name := range categories[1:] {
		if name == category {
			return true
		}
	}
	return false
}
func (m *model) savePreferences() {
	if m.statePath == "" {
		return
	}
	p := preferences{Types: map[string]savedType{}}
	builtin := map[string]bool{}
	for _, kind := range catalog {
		builtin[kind.ext] = true
	}
	for _, r := range m.rows {
		name := ""
		if !builtin[r.kind.ext] {
			name = r.kind.name
		}
		p.Types[r.kind.ext] = savedType{r.kind.category, name, r.modified}
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err == nil {
		err = os.MkdirAll(filepath.Dir(m.statePath), 0700)
	}
	if err == nil {
		var f *os.File
		f, err = os.CreateTemp(filepath.Dir(m.statePath), "state-*.json")
		if err == nil {
			name := f.Name()
			defer os.Remove(name)
			_, err = f.Write(data)
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
			if err == nil {
				err = os.Rename(name, m.statePath)
			}
		}
	}
	if err != nil {
		m.notice += "\n保存本机配置失败：" + err.Error()
	}
}
func (m *model) switchCategory(index int) {
	if index < 0 || index >= len(categories) || m.busy || index == m.category {
		return
	}
	m.category = index
	m.notice = ""
	for i := range m.rows {
		m.rows[i].result = ""
		m.rows[i].outcome = 0
	}
	m.view.ScrollState(id("file-list")).ScrollTo(0)
}
func (m *model) moveType(ext, category string) {
	if m.busy || !validCategory(category) {
		return
	}
	for i := range m.rows {
		if m.rows[i].kind.ext == ext {
			m.rows[i].kind.category = category
			m.savePreferences()
			return
		}
	}
}
func (m *model) recordResults(before, after []row, results []association.Result, undo bool, at time.Time) {
	successfulTypes := map[string]bool{}
	succeeded := 0
	for _, result := range results {
		message := result.Message
		outcome := 1
		if result.Err != nil {
			outcome = 2
			message = "失败：" + result.Err.Error()
			if result.Message != "" {
				message += "；" + result.Message
			}
		} else {
			succeeded++
		}
		for i := range after {
			if after[i].kind.ext == result.Extension {
				after[i].result = message
				after[i].outcome = outcome
				if outcome == 1 {
					successfulTypes[after[i].typ.ID] = true
				}
			}
		}
	}
	for i := range after {
		if successfulTypes[after[i].typ.ID] && (before[i].typ.Handlers != after[i].typ.Handlers || !strings.EqualFold(before[i].typ.Default.ID, after[i].typ.Default.ID)) {
			after[i].modified = at
		}
	}
	m.rows = after
	m.canUndo = m.service.CanUndo()
	action := "修改"
	if undo {
		action = "撤销"
	}
	m.notice = fmt.Sprintf("%s完成：%d 项成功，%d 项失败。", action, succeeded, len(results)-succeeded)
	m.savePreferences()
}
func (m *model) visibleRows() []int {
	query := strings.ToLower(strings.TrimSpace(m.search.Text()))
	var visible []int
	for i, r := range m.rows {
		if m.category > 0 && r.kind.category != categories[m.category] {
			continue
		}
		if query == "" || strings.Contains(strings.ToLower("."+r.kind.ext+" "+r.kind.name+" "+r.kind.category+" "+r.typ.Default.Name), query) {
			visible = append(visible, i)
		}
	}
	sort.SliceStable(visible, func(i, j int) bool {
		a, b := m.rows[visible[i]], m.rows[visible[j]]
		if !a.modified.Equal(b.modified) {
			return a.modified.After(b.modified)
		}
		an, bn := strings.ToLower(a.kind.name), strings.ToLower(b.kind.name)
		if an != bn {
			return an < bn
		}
		return a.kind.ext < b.kind.ext
	})
	return visible
}
