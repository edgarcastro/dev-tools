package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var entry = HookEntry{
	Event:   "PreToolUse",
	Matcher: "Read|Bash",
	Command: "/home/u/.local/bin/block-env-hook",
	Markers: []string{"block-env-hook", "block-env.sh"},
}

func read(t *testing.T, p string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]any{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestAddHookIdempotentAndPreservesKeys(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`{"model":"sonnet","hooks":{"PreToolUse":[{"matcher":"X","hooks":[{"type":"command","command":"other"}]}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, backup, err := AddHook(p, entry)
	if err != nil || !changed || backup == "" {
		t.Fatalf("first add: changed=%v backup=%q err=%v", changed, backup, err)
	}
	changed, _, err = AddHook(p, entry)
	if err != nil || changed {
		t.Fatalf("second add should be a no-op: changed=%v err=%v", changed, err)
	}
	root := read(t, p)
	if root["model"] != "sonnet" {
		t.Error("model key lost")
	}
	if n := len(root["hooks"].(map[string]any)["PreToolUse"].([]any)); n != 2 {
		t.Errorf("want 2 groups (other + ours), got %d", n)
	}
}

func TestAddHookReplacesLegacy(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	legacy := `{"hooks":{"PreToolUse":[{"matcher":"Read","hooks":[{"type":"command","command":"\"$HOME\"/.claude/hooks/block-env.sh"}]}]}}`
	if err := os.WriteFile(p, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AddHook(p, entry); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if strings.Contains(string(b), "block-env.sh") || strings.Count(string(b), "block-env-hook") != 1 {
		t.Errorf("legacy not replaced cleanly:\n%s", b)
	}
}

func TestRemoveHookOnlyOurs(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(p, []byte(`{"hooks":{"PreToolUse":[{"matcher":"X","hooks":[{"type":"command","command":"other"}]}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AddHook(p, entry); err != nil {
		t.Fatal(err)
	}
	changed, _, err := RemoveHook(p, entry)
	if err != nil || !changed {
		t.Fatalf("remove: changed=%v err=%v", changed, err)
	}
	if HasHook(p, entry) {
		t.Error("hook still present")
	}
	if n := len(read(t, p)["hooks"].(map[string]any)["PreToolUse"].([]any)); n != 1 {
		t.Errorf("other hook should remain, groups=%d", n)
	}
}

func TestAddHookCreatesMissingFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nested", "settings.json")
	if changed, backup, err := AddHook(p, entry); err != nil || !changed || backup != "" {
		t.Fatalf("changed=%v backup=%q err=%v", changed, backup, err)
	}
	if !HasHook(p, entry) {
		t.Error("hook missing")
	}
}
