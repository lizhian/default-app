// Package association owns the query/change/undo boundary for file associations.
package association

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type Application struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}
type Type struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Extensions []string    `json:"extensions"`
	Handlers   [4]string   `json:"handlers"` // viewer, editor, shell, all roles
	Default    Application `json:"default"`
}

// System is the external macOS boundary. Set changes a role preference (0–3).
type System interface {
	Resolve(extension string) (Type, error)
	Set(typeID string, role int, applicationID string) error
	Applications(typeID string) ([]Application, error)
}
type Result struct {
	Extension string
	Message   string
	Err       error
}
type snapshot struct {
	extension string
	before    Type
}
type Service struct {
	system  System
	history []snapshot
}

func New(system System) *Service { return &Service{system: system} }

func Normalize(extension string) (string, error) {
	ext := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(extension), "."))
	if ext == "" || len(ext) > 100 {
		return "", errors.New("请输入有效后缀，例如 md；无需输入文件名")
	}
	for _, r := range ext {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '+' {
			return "", errors.New("后缀只能包含字母、数字、连字符、下划线或加号")
		}
	}
	return ext, nil
}
func (s *Service) Query(ext string) (Type, error) {
	ext, err := Normalize(ext)
	if err != nil {
		return Type{}, err
	}
	return s.system.Resolve(ext)
}
func (s *Service) Applications(ext string) ([]Application, error) {
	typ, err := s.Query(ext)
	if err != nil {
		return nil, err
	}
	return s.system.Applications(typ.ID)
}
func (s *Service) CanUndo() bool { return len(s.history) > 0 }
func (s *Service) Apply(extensions []string, appID string) []Result {
	results := make([]Result, 0, len(extensions))
	if appID == "" {
		for _, ext := range extensions {
			results = append(results, Result{Extension: ext, Err: errors.New("请先选择应用")})
		}
		return results
	}
	var history []snapshot
	seen := map[string]Result{}
	for _, raw := range extensions {
		r := Result{Extension: raw}
		ext, err := Normalize(raw)
		if err != nil {
			r.Err = err
			results = append(results, r)
			continue
		}
		r.Extension = ext
		before, err := s.Query(ext)
		if err != nil {
			r.Err = err
			results = append(results, r)
			continue
		}
		if previous, ok := seen[before.ID]; ok {
			previous.Extension = ext
			results = append(results, previous)
			continue
		}
		if sameHandlers(before.Handlers, [4]string{appID, appID, appID, appID}) && strings.EqualFold(before.Default.ID, appID) {
			r.Message = "已是目标应用，无需修改"
			seen[before.ID] = r
			results = append(results, r)
			continue
		}
		// Keep the snapshot even if the native call fails or readback is inconclusive:
		// a multi-role operation may have partially changed the OS.
		changed := false
		for role := range before.Handlers {
			if strings.EqualFold(before.Handlers[role], appID) {
				continue
			}
			if err := s.system.Set(before.ID, role, appID); err != nil {
				r.Err = fmt.Errorf("修改未完成：%w", err)
				break
			}
			changed = true
		}
		if changed {
			history = append(history, snapshot{ext, before})
		}
		after, readErr := s.Query(ext)
		if r.Err == nil {
			if readErr != nil {
				r.Err = fmt.Errorf("修改后无法核实：%w", readErr)
			} else if !sameHandlers(after.Handlers, [4]string{appID, appID, appID, appID}) || !strings.EqualFold(after.Default.ID, appID) {
				r.Err = errors.New("系统读回的默认应用与目标不一致，请刷新后重试")
			} else {
				r.Message = "已修改"
			}
		}
		if changed {
			for _, old := range before.Handlers {
				if old == "" {
					r.Message += "；原先部分角色没有默认应用，无法完整撤销"
					break
				}
			}
		}
		seen[before.ID] = r
		results = append(results, r)
	}
	if len(history) > 0 {
		s.history = history
	}
	return results
}
func (s *Service) Undo() []Result {
	results := make([]Result, 0, len(s.history))
	var remaining []snapshot
	for _, entry := range s.history {
		r := Result{Extension: entry.extension, Message: "已恢复原默认应用"}
		var errs []error
		for role, appID := range entry.before.Handlers {
			if appID == "" {
				errs = append(errs, errors.New("原先没有默认应用，macOS 不提供清除此关联的公开接口"))
				continue
			}
			if err := s.system.Set(entry.before.ID, role, appID); err != nil {
				errs = append(errs, err)
			}
		}
		after, err := s.Query(entry.extension)
		if err != nil {
			errs = append(errs, err)
		} else if !sameHandlers(after.Handlers, entry.before.Handlers) || !strings.EqualFold(after.Default.ID, entry.before.Default.ID) {
			errs = append(errs, errors.New("系统读回的关联尚未恢复"))
		}
		if len(errs) > 0 {
			r.Err = errors.Join(errs...)
			r.Message = "恢复未完成"
			remaining = append(remaining, entry)
		}
		results = append(results, r)
	}
	s.history = remaining
	return results
}

func sameHandlers(a, b [4]string) bool {
	for i := range a {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}
