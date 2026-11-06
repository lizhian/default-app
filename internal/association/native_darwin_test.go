package association_test

import (
	"fmt"
	"local/defaultapp/internal/association"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNativeChangeAndUndoTemporaryType(t *testing.T) {
	if os.Getenv("DEFAULTAPP_NATIVE_TEST") != "1" {
		t.Skip("opt in with DEFAULTAPP_NATIVE_TEST=1; registers temporary test bundles only")
	}
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	ext := "defaultapptest" + token
	typeID := "dev.defaultapp.test.t" + token
	if err := os.MkdirAll("../../work", 0755); err != nil {
		t.Fatal(err)
	}
	dir, tempErr := os.MkdirTemp("../../work", "native-test-")
	if tempErr != nil {
		t.Fatal(tempErr)
	}
	dir, _ = filepath.Abs(dir)
	t.Cleanup(func() { os.RemoveAll(dir) })
	makeApp := func(name string) association.Application {
		path := filepath.Join(dir, name+".app")
		if err := os.MkdirAll(filepath.Join(path, "Contents", "MacOS"), 0755); err != nil {
			t.Fatal(err)
		}
		id := "dev.defaultapp.test." + strings.ToLower(name) + token
		plist := fmt.Sprintf(`<?xml version="1.0"?><plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>%s</string><key>CFBundleName</key><string>%s</string>
<key>CFBundleVersion</key><string>1</string><key>CFBundleShortVersionString</key><string>1.0</string><key>CFBundleExecutable</key><string>fixture</string><key>CFBundlePackageType</key><string>APPL</string>
<key>UTExportedTypeDeclarations</key><array><dict><key>UTTypeIdentifier</key><string>%s</string><key>UTTypeConformsTo</key><array><string>public.data</string></array><key>UTTypeTagSpecification</key><dict><key>public.filename-extension</key><array><string>%s</string></array></dict></dict></array>
<key>CFBundleDocumentTypes</key><array><dict><key>CFBundleTypeName</key><string>Temporary test</string><key>CFBundleTypeRole</key><string>Editor</string><key>LSItemContentTypes</key><array><string>%s</string></array></dict></array>
</dict></plist>`, id, name, typeID, ext, typeID)
		if err := os.WriteFile(filepath.Join(path, "Contents", "Info.plist"), []byte(plist), 0644); err != nil {
			t.Fatal(err)
		}
		binary, readErr := os.ReadFile("/usr/bin/true")
		if readErr != nil {
			t.Fatal(readErr)
		}
		if err := os.WriteFile(filepath.Join(path, "Contents", "MacOS", "fixture"), binary, 0755); err != nil {
			t.Fatal(err)
		}
		if err := association.RegisterApplication(path); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = exec.Command("/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister", "-u", path).Run()
		})
		app, err := association.ApplicationAtPath(path)
		if err != nil {
			t.Fatal(err)
		}
		return app
	}
	a, b := makeApp("Before"), makeApp("After")
	native := association.Native{}
	// Seed only this unique, temporary type so every role has a restorable value.
	if err := native.Set(typeID, 3, a.ID); err != nil {
		t.Fatal(err)
	}

	service := association.New(native)
	before, err := service.Query(ext)
	if err != nil || before.ID != typeID {
		t.Fatalf("temporary type not registered: %+v %v", before, err)
	}
	candidates, err := service.Applications(ext)
	if err != nil || len(candidates) < 2 {
		t.Fatalf("missing applications: %+v %v", candidates, err)
	}
	result := service.Apply([]string{ext}, b.ID)
	if result[0].Err != nil {
		t.Fatal(result[0].Err)
	}
	changed, err := service.Query(ext)
	if err != nil || !strings.EqualFold(changed.Default.ID, b.ID) {
		t.Fatalf("not changed: %+v %v", changed, err)
	}
	result = service.Undo()
	if len(result) != 1 || result[0].Err != nil {
		t.Fatalf("undo: %+v", result)
	}
	restored, err := service.Query(ext)
	if err != nil || restored.Handlers != before.Handlers || !strings.EqualFold(restored.Default.ID, before.Default.ID) {
		t.Fatalf("not restored: %+v %v", restored, err)
	}
}
