package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
int defaultapp_application_menu(const char *items, const char *selected, double x, double y, double width, double height);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"github.com/vivaui/viva/ui"
	"local/defaultapp/internal/association"
	"unsafe"
)

// Open the system menu after Paint finishes, anchored to the laid-out trigger.
func (m *model) showApplicationMenu() {
	if m.busy || len(m.selected()) == 0 || len(m.apps) == 0 {
		return
	}
	picked, err := pickMenu(m.apps, m.target.ID, m.menuRect)
	if err != nil {
		m.notice = err.Error()
	}
	if picked >= 0 && picked < len(m.apps) {
		m.target = m.apps[picked]
	}
}
func pickMenu(apps []association.Application, selectedID string, r ui.Rect) (int, error) {
	items, err := json.Marshal(apps)
	if err != nil {
		return -1, err
	}
	data, selected := C.CString(string(items)), C.CString(selectedID)
	defer C.free(unsafe.Pointer(data))
	defer C.free(unsafe.Pointer(selected))
	picked := int(C.defaultapp_application_menu(data, selected, C.double(r.X), C.double(r.Y), C.double(r.Width), C.double(r.Height)))
	if picked < -1 {
		return picked, fmt.Errorf("无法显示菜单（%d），请重试", picked)
	}
	return picked, nil
}
