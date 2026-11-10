package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Cocoa -framework CoreServices
#include <stdlib.h>
char *defaultapp_preview_candidates(const char *paths, const char *types);
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
	"unsafe"

	"github.com/vivaui/viva/flex"
	"local/defaultapp/internal/association"
)

func previewCandidates(types []string) (map[string][]association.Application, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "/usr/bin/pluginkit", "-m", "-v", "-p", "com.apple.quicklook.preview").Output()
	if err != nil {
		return nil, fmt.Errorf("无法查询预览扩展：%w", err)
	}
	paths := []string{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue
		}
		path := strings.TrimSpace(fields[len(fields)-1])
		if strings.HasPrefix(path, "/") && strings.HasSuffix(path, ".appex") {
			paths = append(paths, path)
		}
	}
	return candidatesForPaths(paths, types)
}

func candidatesForPaths(paths, types []string) (map[string][]association.Application, error) {
	p, _ := json.Marshal(paths)
	t, _ := json.Marshal(types)
	cp, ct := C.CString(string(p)), C.CString(string(t))
	defer C.free(unsafe.Pointer(cp))
	defer C.free(unsafe.Pointer(ct))
	raw := C.defaultapp_preview_candidates(cp, ct)
	if raw == nil {
		return nil, fmt.Errorf("无法读取预览扩展声明")
	}
	defer C.free(unsafe.Pointer(raw))
	result := map[string][]association.Application{}
	err := json.Unmarshal([]byte(C.GoString(raw)), &result)
	for key := range result {
		sort.Slice(result[key], func(i, j int) bool {
			return strings.ToLower(result[key][i].Name) < strings.ToLower(result[key][j].Name)
		})
	}
	return result, err
}

func (m *model) previewCell(f *flex.Frame, r row) {
	label := "未识别"
	if r.previewErr != "" {
		label = "查询失败"
	}
	if len(r.previews) > 0 {
		label = r.previews[0].Name
		if len(r.previews) > 1 {
			label += fmt.Sprintf(" +%d", len(r.previews)-1)
		}
	}
	cell := f.Row().Width(156).Gap(6).Items(flex.Center)
	if !m.busy {
		cell = cell.Hit(id("preview-"+r.kind.ext), "查看 ."+r.kind.ext+" 预览候选")
	}
	cell.Children(func() {
		if len(r.previews) > 0 {
			m.appIcon(f, r.previews[0].Path)
		}
		f.Text(label).Grow(1).Ellipsis()
	})
	if cell.Clicked() {
		m.window.Post(func() {
			items := []association.Application{{Name: "Quick Look 候选，不代表实际生效"}}
			if r.previewErr != "" {
				items = append(items, association.Application{Name: r.previewErr})
			}
			if len(r.previews) == 0 {
				items = append(items, association.Application{Name: "未识别；系统仍可能支持预览"})
			}
			for _, a := range r.previews {
				a.Name += " · " + a.ID
				items = append(items, a)
			}
			if _, err := pickMenu(items, "", cell.Rect()); err != nil {
				m.notice = err.Error()
			}
		})
	}
}
