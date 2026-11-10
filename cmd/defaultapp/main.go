package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/flex"
	"github.com/vivaui/viva/ui"
	"local/defaultapp/internal/association"
)

var tree ui.Tree
var page = tree.Feature("default-app")

func id(name string) ui.ID { return tree.ID(page, name) }

var categoryScope = tree.Feature("categories")

type row struct {
	kind     fileKind
	typ      association.Type
	err      error
	selected bool
	result   string
	outcome  int
	modified time.Time
}
type model struct {
	view                    flex.View
	search, custom, appPath flex.Field
	appAnchor               flex.Box
	menuRect                ui.Rect
	table                   flex.Box
	category                int
	icons                   sync.Map
	nextIcon                atomic.Uint64
	service                 *association.Service
	window                  *app.ContentWindow
	rows                    []row
	apps                    []association.Application
	target                  association.Application
	busy, canUndo           bool
	afterAdd                bool
	pathExpanded            bool
	notice                  string
	closeAfter              func()
	statePath               string
}

func main() {
	m := newModel(association.New(association.Native{}), defaultStatePath())

	err := app.RunApplication(func(application *app.Application) error {
		var canvas ui.Canvas
		var err error
		m.window, err = application.OpenContentWindow(m.config(), &canvas)
		if err == nil {
			m.refresh()
		}
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
}

// All system operations run serially off the UI thread. Only Post updates UI state.
func (m *model) run(work func() func()) {
	if m.busy {
		return
	}
	m.busy = true
	go func() {
		update := work()
		m.window.Post(func() {
			update()
			m.busy = false
			if m.closeAfter != nil {
				m.closeAfter()
				return
			}
			if m.afterAdd {
				m.afterAdd = false
				m.loadApps()
			}
		})
	}()
}
func (m *model) readRows(rows []row) []row {
	for i := range rows {
		rows[i].typ, rows[i].err = m.service.Query(rows[i].kind.ext)
		m.cacheIcon(rows[i].typ.Default.Path)
	}
	return rows
}
func (m *model) refresh() {
	rows := append([]row(nil), m.rows...)
	m.run(func() func() { rows = m.readRows(rows); return func() { m.rows = rows } })
}
func (m *model) selected() []string {
	var exts []string
	for _, r := range m.rows {
		if r.selected {
			exts = append(exts, r.kind.ext)
		}
	}
	return exts
}
func (m *model) loadApps() {
	exts := m.selected()
	chosen := m.target
	m.run(func() func() {
		byID := map[string]association.Application{}
		var failures []string
		for _, ext := range exts {
			apps, err := m.service.Applications(ext)
			if err != nil {
				failures = append(failures, "."+ext+"："+err.Error())
				continue
			}
			for _, a := range apps {
				byID[strings.ToLower(a.ID)] = a
			}
		}
		if chosen.ID != "" {
			byID[strings.ToLower(chosen.ID)] = chosen
		}
		apps := make([]association.Application, 0, len(byID))
		for _, a := range byID {
			apps = append(apps, a)
		}
		sort.Slice(apps, func(i, j int) bool { return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name) })
		for _, a := range apps {
			m.cacheIcon(a.Path)
		}
		return func() {
			m.apps = apps
			if len(failures) > 0 {
				m.notice = strings.Join(failures, "；")
			}
		}
	})
}
func (m *model) toggle(index int) {
	if m.busy {
		return
	}
	m.rows[index].selected = !m.rows[index].selected
	m.loadApps()
}
func (m *model) addCustom() {
	if m.busy {
		return
	}
	ext, err := association.Normalize(m.custom.Text())
	if err != nil {
		m.notice = err.Error()
		return
	}
	for i := range m.rows {
		if m.rows[i].kind.ext == ext {
			m.rows[i].selected = true
			for tab, category := range categories {
				if category == m.rows[i].kind.category {
					m.switchCategory(tab)
					break
				}
			}
			m.search.SetText("")
			m.custom.SetText("")
			m.loadApps()
			return
		}
	}
	m.search.SetText("")
	m.custom.SetText("")
	category := "其他"
	if m.category > 0 {
		category = categories[m.category]
	}
	m.run(func() func() {
		typ, err := m.service.Query(ext)
		return func() {
			if err != nil {
				m.notice = err.Error()
				return
			}
			m.rows = append(m.rows, row{kind: fileKind{ext, "自定义类型", category}, typ: typ, selected: true})
			m.notice = "已添加 ." + ext + "；可选择应用后修改。"
			m.savePreferences()
			// Schedule candidate loading after this update has released the busy state.
		}
	})
	// Candidate refresh for new rows is handled by the completion queue below.
	m.afterAdd = true
}
func (m *model) chooseApp() {
	if m.busy {
		return
	}
	paths, err := app.OpenFiles(app.OpenFileOptions{Title: "选择默认打开应用", Filters: []app.FileFilter{{Name: "macOS 应用", Pattern: "*.app"}}})
	if err != nil {
		m.notice = "无法选择应用：" + err.Error()
		return
	}
	if len(paths) == 0 {
		return
	}
	m.chooseAppPath(paths[0])
}

func (m *model) chooseAppPath(path string) {
	path = strings.Trim(strings.TrimSpace(path), "\"'")
	if path == "" || m.busy {
		return
	}
	m.run(func() func() {
		a, err := association.ApplicationAtPath(path)
		if err == nil {
			err = association.RegisterApplication(path)
			m.cacheIcon(a.Path)
		}
		return func() {
			if err != nil {
				m.notice = err.Error()
				return
			}
			m.target = a
			found := false
			for _, old := range m.apps {
				if strings.EqualFold(old.ID, a.ID) {
					found = true
				}
			}
			if !found {
				m.apps = append(m.apps, a)
			}
			m.notice = "已选择 " + a.Name + "；点击应用修改后生效。"
		}
	})
}
func (m *model) change(undo bool) { m.applyChange(m.selected(), m.target, undo) }
func (m *model) applyChange(exts []string, target association.Application, undo bool) {
	if m.busy || (!undo && (len(exts) == 0 || target.ID == "")) {
		return
	}
	before := append([]row(nil), m.rows...)
	m.run(func() func() {
		var results []association.Result
		if undo {
			results = m.service.Undo()
		} else {
			results = m.service.Apply(exts, target.ID)
		}
		after := m.readRows(append([]row(nil), before...))
		at := time.Now().UTC()
		return func() { m.recordResults(before, after, results, undo, at) }
	})
}
func (m *model) config() app.Config {
	theme := appTheme()
	config := app.Config{Tree: &tree, Title: "默认打开方式", Width: 980, Height: 700, MinWidth: 880, MinHeight: 620, WindowMode: app.WindowCustomTitlebar, TitlebarHeight: 56, TextInput: flex.Fields(&m.search, &m.custom, &m.appPath)}
	leading, trailing := app.TitlebarInsets()
	config.OnCloseRequest = func(allow func()) {
		if m.busy {
			m.closeAfter = allow
			m.notice = "正在完成操作，完成后关闭。"
		} else {
			allow()
		}
	}
	config.OnScroll = m.view.Wheel

	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		canvas.Fill(ui.Rect{Width: size.Width, Height: size.Height}, theme.Background)
		m.view.Draw(canvas, theme, ui.Rect{Width: size.Width, Height: size.Height}, func(f *flex.Frame) {
			f.Column().Fill().Children(func() {
				f.Row().Height(56).PadSides(0, trailing+20, 0, leading+12).Items(flex.Center).Gap(12).MovesWindow().Children(func() {
					if index, ok := f.Tabs(categoryScope, categories, m.category).Grow(1).Disabled(m.busy).TabClicked(); ok {
						m.switchCategory(index)
					}
					f.Slot().Width(8)
					f.Field(&m.search, id("search"), "搜索文件类型或默认应用", "搜索后缀、名称、分类或应用").Width(240)
					if f.Button(id("refresh"), "刷新").Outline().Disabled(m.busy).Clicked() {
						m.refresh()
					}
				})
				f.Divider()
				f.Row().Grow(1).Pad(20).Gap(24).Children(func() {
					f.Column().Grow(1).Gap(12).Children(func() {
						m.fileList(f, theme)
						f.Row().Items(flex.Center).Gap(8).Children(func() {
							submitted := f.Field(&m.custom, id("custom"), "自定义后缀", "自定义后缀，如 md").Grow(1).Disabled(m.busy).Submitted()
							if f.Button(id("add"), "添加后缀").Outline().Disabled(m.busy).Clicked() || submitted {
								m.addCustom()
							}
							if f.Button(id("select-visible"), "全选结果").Ghost().Disabled(m.busy || len(m.visibleRows()) == 0).Clicked() {
								m.selectVisibleRows()
								m.loadApps()
							}
							if f.Button(id("clear"), "取消选择").Ghost().Disabled(m.busy || len(m.selected()) == 0).Clicked() {
								for i := range m.rows {
									m.rows[i].selected = false
								}
								m.loadApps()
							}
						})
					})
					f.Column().Width(264).Gap(12).Children(func() { m.applicationPanel(f) })
				})
			})
		})
		m.menuRect = m.appAnchor.Rect()
		paintTableCorners(canvas, m.table.Rect(), theme)
	}
	return config
}
func (m *model) fileList(f *flex.Frame, theme ui.Theme) {
	visible := m.visibleRows()

	m.table = f.Column().Grow(1).Background(theme.Card).Radius(8).Clip()
	m.table.Children(func() {
		f.Row().Height(32).PadX(12).Items(flex.Center).Gap(10).Background(theme.Muted).Children(func() {
			f.Slot().Width(24)
			f.Text("后缀").Width(66).FontSize(11).Muted()
			f.Text("名称").Grow(1).FontSize(11).Muted()
			f.Text("分类").Width(72).FontSize(11).Muted()
			f.Text("当前默认应用").Width(168).FontSize(11).Muted()
		})
		f.Divider()
		f.ScrollArea(id("file-list")).Grow(1).Children(func() {
			if m.busy && len(m.rows) > 0 && m.rows[0].typ.ID == "" {
				f.Paragraph("正在读取文件类型…", 2).Pad(16).Muted()
				return
			}
			if len(visible) == 0 {
				f.Paragraph("没有匹配的文件类型。可在下方输入后缀并添加。", 3).Pad(16).Muted()
			}
			for _, i := range visible {
				r := m.rows[i]
				bg := theme.Card
				if r.selected {
					bg = theme.Accent
				}
				if r.outcome == 1 {
					bg = theme.Success.WithAlpha(0.12).Over(theme.Card)
				}
				if r.outcome == 2 {
					bg = theme.Danger.WithAlpha(0.10).Over(theme.Card)
				}
				container := f.Column().PadX(12).PadY(7).Gap(4).Background(bg)
				if !m.busy {
					container = container.Hit(id("row-"+r.kind.ext), "选择 ."+r.kind.ext)
				}
				container.Children(func() {
					f.Row().MinHeight(32).Items(flex.Center).Gap(10).Children(func() {
						if f.Checkbox(id("ext-"+r.kind.ext), r.selected).Width(24).Label("选择 ." + r.kind.ext).Disabled(m.busy).Clicked() {
							m.toggle(i)
						}
						f.Text("." + r.kind.ext).Width(66).Weight(ui.WeightMedium).Ellipsis()
						f.Text(r.kind.name).Grow(1).Ellipsis()
						categoryButton := f.Row().Width(72).MinHeight(32).Items(flex.Center)
						if !m.busy {
							categoryButton = categoryButton.Hit(id("category-"+r.kind.ext), r.kind.category)
						}
						categoryButton.Children(func() {
							f.Text(r.kind.category).FontSize(12)
						})
						if categoryButton.Clicked() {
							ext := r.kind.ext
							m.window.Post(func() { m.editCategory(ext, categoryButton.Rect()) })
						}
						name := r.typ.Default.Name
						if r.err != nil {
							name = "读取失败"
						} else if name == "" {
							name = "未设置"
						}
						appCell := f.Row().Width(168).Gap(6).Items(flex.Center)
						if !m.busy {
							appCell = appCell.Hit(id("current-app-"+r.kind.ext), "修改 ."+r.kind.ext+" 默认应用")
						}
						appCell.Children(func() {
							m.appIcon(f, r.typ.Default.Path)
							f.Text(name).Grow(1).Ellipsis()
						})
						if appCell.Clicked() {
							ext := r.kind.ext
							m.window.Post(func() { m.quickApplication(ext, appCell.Rect()) })
						}
					})
				})
				if container.Clicked() {
					m.toggle(i)
				}
				f.Divider()
			}
		})
	})
}
func (m *model) applicationPanel(f *flex.Frame) {
	selected := m.selected()
	f.Row().Height(24).Items(flex.Center).Gap(8).Children(func() {
		f.Text(fmt.Sprintf("已选择 %d 种后缀", len(selected))).FontSize(15).Bold()
		if m.busy {
			f.Spinner()
		}
	})
	f.ScrollArea(id("scope-notes")).Grow(1).Gap(12).Children(func() {
		labels := make([]string, len(selected))
		for i, ext := range selected {
			labels[i] = "." + ext
		}
		text := "在左侧勾选要修改的文件类型。"
		if len(labels) > 0 {
			text = strings.Join(labels, "、")
		}
		f.Paragraph(text, 4).FontSize(12).LineHeight(17).Muted()
		f.Divider()
		f.Text("默认打开应用").Weight(ui.WeightMedium)
		if len(selected) == 0 {
			f.Wrap("选择文件类型后，会列出系统推荐的应用。").FontSize(12).LineHeight(17).Muted()
		}
		f.Row().Gap(8).Items(flex.Center).Children(func() {
			if m.target.Path != "" {
				m.appIcon(f, m.target.Path)
			}
			label := "选择应用"
			if m.target.ID != "" {
				label = m.target.Name
			}
			m.appAnchor = f.Button(id("application-menu"), label+"  ⌄").Outline().Grow(1).Disabled(m.busy || len(selected) == 0 || len(m.apps) == 0)
			if m.appAnchor.Clicked() {
				m.window.Post(m.showApplicationMenu)
			}
		})
		if len(selected) > 1 {
			f.Wrap("推荐应用可能只支持部分所选类型。").FontSize(12).LineHeight(17).Muted()
		}
		f.Row().Gap(8).Items(flex.Center).Children(func() {
			if f.Button(id("choose-app"), "选择其他应用…").Outline().Disabled(m.busy).Clicked() {
				m.window.Post(m.chooseApp)
			}
			disclosure := "输入应用路径…"
			if m.pathExpanded {
				disclosure = "收起应用路径"
			}
			if f.Button(id("toggle-path"), disclosure).Ghost().Disabled(m.busy).Clicked() {
				m.pathExpanded = !m.pathExpanded
			}
		})
		if m.pathExpanded {
			f.Field(&m.appPath, id("application-path"), "应用完整路径", "粘贴 .app 的完整路径").Disabled(m.busy)
			if f.Button(id("use-path"), "使用此路径").Outline().Disabled(m.busy || strings.TrimSpace(m.appPath.Text()) == "").Clicked() {
				m.chooseAppPath(m.appPath.Text())
			}
		}
		if m.pathExpanded && m.target.Path != "" {
			f.Wrap(m.target.Path).FontSize(11).Muted()
		}
		f.Divider()
		f.Text("修改范围").Weight(ui.WeightMedium)
		if m.notice != "" {
			f.Wrap(m.notice).FontSize(12).LineHeight(17).Muted()
		}
		for _, r := range m.rows {
			if r.result != "" {
				f.Wrap("." + r.kind.ext + "：" + r.result).FontSize(12).LineHeight(17).Muted()
			}
		}
		if len(selected) > 0 {
			seen := map[string]bool{}
			missingDefault := false
			for _, r := range m.rows {
				if !r.selected {
					continue
				}
				for _, handler := range r.typ.Handlers {
					if handler == "" {
						missingDefault = true
					}
				}
				if seen[r.typ.ID] {
					continue
				}
				seen[r.typ.ID] = true
				extensions := r.typ.Extensions
				if len(extensions) == 0 {
					extensions = []string{r.kind.ext}
				}
				items := make([]string, len(extensions))
				for i, ext := range extensions {
					items[i] = "." + ext
				}
				f.Wrap(strings.Join(items, "、")).FontSize(12).LineHeight(17).Muted()
			}
			f.Wrap("同一系统文件类型的其他后缀也会受影响。仅保留本次启动中的上一步撤销记录。").FontSize(12).LineHeight(17).Muted()
			if missingDefault {
				f.Wrap("部分类型原先未设置默认应用，修改后可能无法完整撤销。").FontSize(12).LineHeight(17).Muted()
			}
		}
	})
	f.Divider()
	f.Row().Gap(8).Items(flex.Center).Children(func() {
		if f.Button(id("undo"), "撤销上次修改").Outline().Grow(1).Disabled(m.busy || !m.canUndo).Clicked() {
			m.change(true)
		}
		if f.Button(id("apply"), "应用修改").Width(104).Disabled(m.busy || len(selected) == 0 || m.target.ID == "").Clicked() {
			m.change(false)
		}
	})
}
