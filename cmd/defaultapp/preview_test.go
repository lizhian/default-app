package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewCandidatesMatchDeclaredTypeAndInheritance(t *testing.T) {
	root := t.TempDir()
	paths := []string{}
	for _, name := range []string{"One", "Two"} {
		path := filepath.Join(root, name+".app", "Contents", "PlugIns", "Preview.appex")
		if err := os.MkdirAll(filepath.Join(path, "Contents"), 0700); err != nil {
			t.Fatal(err)
		}
		plist := `<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>test.` + name + `</string><key>CFBundlePackageType</key><string>XPC!</string><key>NSExtension</key><dict><key>NSExtensionAttributes</key><dict><key>QLSupportedContentTypes</key><array><string>public.text</string></array></dict></dict></dict></plist>`
		if err := os.WriteFile(filepath.Join(path, "Contents", "Info.plist"), []byte(plist), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	// Duplicate registration must not create a duplicate candidate.
	paths = append(paths, paths[0])
	got, err := candidatesForPaths(paths, []string{"public.text", "public.json", "public.png", ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(got["public.text"]) != 2 || len(got["public.json"]) != 2 || len(got["public.png"]) != 0 || len(got[""]) != 0 {
		t.Fatalf("incorrect matching: %+v", got)
	}
	if got["public.json"][0].Path != filepath.Join(root, "One.app") {
		t.Fatalf("missing owning app: %+v", got)
	}
}
