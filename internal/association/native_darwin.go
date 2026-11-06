package association

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Cocoa -framework CoreServices
#include <stdlib.h>
char *da_resolve(const char *extension);
char *da_applications(const char *type);
char *da_application(const char *path);
int da_set(const char *type, int role, const char *app);
int da_register(const char *path);
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"
)

type Native struct{}

func decodeNative[T any](raw *C.char) (T, error) {
	var value T
	if raw == nil {
		return value, fmt.Errorf("macOS 无法读取此类型或应用")
	}
	defer C.free(unsafe.Pointer(raw))
	err := json.Unmarshal([]byte(C.GoString(raw)), &value)
	return value, err
}
func (Native) Resolve(ext string) (Type, error) {
	c := C.CString(ext)
	defer C.free(unsafe.Pointer(c))
	return decodeNative[Type](C.da_resolve(c))
}
func (Native) Applications(id string) ([]Application, error) {
	c := C.CString(id)
	defer C.free(unsafe.Pointer(c))
	return decodeNative[[]Application](C.da_applications(c))
}

// Set accepts viewer (0), editor (1), shell (2), or all roles (3).
func (Native) Set(id string, role int, app string) error {
	a, b := C.CString(id), C.CString(app)
	defer C.free(unsafe.Pointer(a))
	defer C.free(unsafe.Pointer(b))
	status := C.da_set(a, C.int(role), b)
	if status != 0 {
		return fmt.Errorf("macOS 拒绝修改（OSStatus %d）；请确认应用仍然存在", int(status))
	}
	return nil
}
func ApplicationAtPath(path string) (Application, error) {
	c := C.CString(path)
	defer C.free(unsafe.Pointer(c))
	return decodeNative[Application](C.da_application(c))
}

// RegisterApplication registers a chosen bundle without changing any defaults.
func RegisterApplication(path string) error {
	c := C.CString(path)
	defer C.free(unsafe.Pointer(c))
	if status := C.da_register(c); status != 0 {
		return fmt.Errorf("无法注册应用（OSStatus %d）", int(status))
	}
	return nil
}
