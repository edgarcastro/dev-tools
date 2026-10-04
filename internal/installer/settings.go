package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HookEntry is the hook to manage inside a settings file.
type HookEntry struct {
	Event, Matcher, Command string
	Markers                 []string // commands containing any marker are "ours" (replaced / removed)
}

func loadSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	root := map[string]any{}
	if len(strings.TrimSpace(string(data))) == 0 {
		return root, nil
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	return root, nil
}

// writeSettings backs up an existing file, then writes atomically.
func writeSettings(path string, root map[string]any) (backup string, err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if old, err := os.ReadFile(path); err == nil {
		backup = fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102-150405.000"))
		if err := os.WriteFile(backup, old, 0o600); err != nil {
			return "", err
		}
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return backup, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0o644); err != nil {
		return backup, err
	}
	return backup, os.Rename(tmp, path)
}

func isOurs(cmd string, markers []string) bool {
	for _, m := range markers {
		if strings.Contains(cmd, m) {
			return true
		}
	}
	return false
}

// groups returns settings.hooks[event] as a slice (nil if absent).
func groups(root map[string]any, event string) []any {
	hooks, _ := root["hooks"].(map[string]any)
	arr, _ := hooks[event].([]any)
	return arr
}

func setGroups(root map[string]any, event string, arr []any) {
	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		root["hooks"] = hooks
	}
	if len(arr) == 0 {
		delete(hooks, event)
		if len(hooks) == 0 {
			delete(root, "hooks")
		}
		return
	}
	hooks[event] = arr
}

// stripOurs removes inner hooks matching markers, dropping emptied groups.
// It reports how many inner hooks were removed and how many exact matches
// (same matcher + command) were seen.
func stripOurs(arr []any, e HookEntry) (kept []any, removed int, exact int) {
	for _, g := range arr {
		gm, ok := g.(map[string]any)
		if !ok {
			kept = append(kept, g)
			continue
		}
		inner, _ := gm["hooks"].([]any)
		var keepInner []any
		for _, h := range inner {
			hm, _ := h.(map[string]any)
			cmd, _ := hm["command"].(string)
			if isOurs(cmd, e.Markers) {
				removed++
				if cmd == e.Command && gm["matcher"] == e.Matcher {
					exact++
				}
				continue
			}
			keepInner = append(keepInner, h)
		}
		if len(keepInner) == 0 && len(inner) > 0 {
			continue
		}
		if len(keepInner) != len(inner) {
			gm["hooks"] = keepInner
		}
		kept = append(kept, gm)
	}
	return
}

// AddHook installs the hook idempotently, replacing legacy entries.
func AddHook(path string, e HookEntry) (changed bool, backup string, err error) {
	root, err := loadSettings(path)
	if err != nil {
		return false, "", err
	}
	kept, removed, exact := stripOurs(groups(root, e.Event), e)
	if removed == 1 && exact == 1 {
		return false, "", nil // already installed, nothing else of ours
	}
	kept = append(kept, map[string]any{
		"matcher": e.Matcher,
		"hooks":   []any{map[string]any{"type": "command", "command": e.Command}},
	})
	setGroups(root, e.Event, kept)
	backup, err = writeSettings(path, root)
	return err == nil, backup, err
}

// RemoveHook removes only our entries.
func RemoveHook(path string, e HookEntry) (changed bool, backup string, err error) {
	root, err := loadSettings(path)
	if err != nil {
		return false, "", err
	}
	kept, removed, _ := stripOurs(groups(root, e.Event), e)
	if removed == 0 {
		return false, "", nil
	}
	setGroups(root, e.Event, kept)
	backup, err = writeSettings(path, root)
	return err == nil, backup, err
}

// HasHook reports whether an entry of ours exists.
func HasHook(path string, e HookEntry) bool {
	root, err := loadSettings(path)
	if err != nil {
		return false
	}
	_, removed, _ := stripOurs(groups(root, e.Event), e)
	return removed > 0
}
