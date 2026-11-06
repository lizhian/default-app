package association_test

import (
	"fmt"
	"local/defaultapp/internal/association"
	"strings"
	"testing"
)

// The fake replaces only the OS boundary, not any application modules.
type system struct {
	types    map[string]association.Type
	failures map[string]bool
}

func (s *system) Resolve(ext string) (association.Type, error) {
	v, ok := s.types[ext]
	if !ok {
		return v, fmt.Errorf("unknown extension")
	}
	v.Default.ID = v.Handlers[3]
	return v, nil
}
func (s *system) Set(id string, role int, app string) error {
	if s.failures[id] {
		return fmt.Errorf("OS rejected change")
	}
	for ext, v := range s.types {
		if v.ID == id {
			v.Handlers[role] = app
			s.types[ext] = v
		}
	}
	return nil
}
func (*system) Applications(string) ([]association.Application, error) { return nil, nil }
func TestBatchChangeAndUndoPreserveOriginalRoles(t *testing.T) {
	os := &system{types: map[string]association.Type{
		"txt":  {ID: "public.text", Handlers: [4]string{"reader", "editor", "shell", "reader"}},
		"text": {ID: "public.text", Handlers: [4]string{"reader", "editor", "shell", "reader"}},
	}}
	service := association.New(os)
	results := service.Apply([]string{".TXT", "text"}, "new-app")
	for _, r := range results {
		if r.Err != nil {
			t.Fatal(r.Err)
		}
	}
	got, err := service.Query("txt")
	if err != nil || got.Handlers != [4]string{"new-app", "new-app", "new-app", "new-app"} {
		t.Fatalf("change not visible: %+v %v", got, err)
	}
	for _, r := range service.Undo() {
		if r.Err != nil {
			t.Fatal(r.Err)
		}
	}
	got, err = service.Query("text")
	if err != nil || got.Handlers != [4]string{"reader", "editor", "shell", "reader"} {
		t.Fatalf("original roles not restored: %+v %v", got, err)
	}
	if service.CanUndo() {
		t.Fatal("successful undo must consume history")
	}
}

func TestPartialFailureIsReportedAndUndoOnlyRestoresChangedTypes(t *testing.T) {
	os := &system{types: map[string]association.Type{
		"ok":  {ID: "test.ok", Handlers: [4]string{"old", "old", "old", "old"}},
		"bad": {ID: "test.bad", Handlers: [4]string{"old", "old", "old", "old"}},
	}, failures: map[string]bool{"test.bad": true}}
	service := association.New(os)
	results := service.Apply([]string{"ok", "bad", "../txt"}, "new")
	if len(results) != 3 || results[0].Err != nil || results[1].Err == nil || results[2].Err == nil {
		t.Fatalf("missing per-item results: %+v", results)
	}
	undone := service.Undo()
	if len(undone) != 1 || undone[0].Extension != "ok" || undone[0].Err != nil {
		t.Fatalf("wrong undo scope: %+v", undone)
	}
}

func TestUnverifiableWriteIsNotReportedAsSuccess(t *testing.T) {
	os := &ignoringSystem{system: system{types: map[string]association.Type{"txt": {ID: "public.text", Handlers: [4]string{"old", "old", "old", "old"}}}}}
	service := association.New(os)
	if results := service.Apply([]string{"txt"}, "new"); results[0].Err == nil {
		t.Fatal("unverified write must not report success")
	}
}

type ignoringSystem struct{ system }

func (*ignoringSystem) Set(string, int, string) error { return nil }

func TestApplicationIdentifiersAreCaseInsensitive(t *testing.T) {
	os := &lowercaseSystem{system: system{types: map[string]association.Type{"txt": {ID: "public.text", Handlers: [4]string{"old", "old", "old", "old"}}}}}
	service := association.New(os)
	if results := service.Apply([]string{"txt"}, "com.example.Reader"); results[0].Err != nil {
		t.Fatal(results[0].Err)
	}
}

type lowercaseSystem struct{ system }

func (s *lowercaseSystem) Set(id string, role int, app string) error {
	return s.system.Set(id, role, strings.ToLower(app))
}
