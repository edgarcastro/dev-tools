// Package installer builds binaries and wires hooks into Claude Code settings.
package installer

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/edgarcastro/dev-tools/internal/catalog"
)

// Scope selects which settings file receives hook entries.
type Scope string

const (
	User    Scope = "user"    // ~/.claude/settings.json
	Project Scope = "project" // <cwd>/.claude/settings.local.json (holds an absolute path, so not committed)
)

const modulePath = "github.com/edgarcastro/dev-tools"

// Env holds the locations the installer works with (overridable in tests).
type Env struct {
	Home     string
	Cwd      string
	RepoRoot string // source tree used for `go build`
}

// NewEnv resolves the real environment, locating the source tree from the
// cwd or the running executable.
func NewEnv() (Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Env{}, err
	}
	cwd, _ := os.Getwd()
	e := Env{Home: home, Cwd: cwd}
	exe, _ := os.Executable()
	for _, start := range []string{cwd, filepath.Dir(exe)} {
		if root := findRepoRoot(start); root != "" {
			e.RepoRoot = root
			break
		}
	}
	return e, nil
}

func findRepoRoot(dir string) string {
	for dir != "/" && dir != "." && dir != "" {
		if b, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil && strings.Contains(string(b), "module "+modulePath) {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return ""
}

func (e Env) BinDir() string { return filepath.Join(e.Home, ".local", "bin") }

func (e Env) SettingsPath(s Scope) string {
	if s == Project {
		return filepath.Join(e.Cwd, ".claude", "settings.local.json")
	}
	return filepath.Join(e.Home, ".claude", "settings.json")
}

func (e Env) hookEntry(h *catalog.Hook) HookEntry {
	return HookEntry{
		Event:   h.Event,
		Matcher: h.Matcher,
		Command: filepath.Join(e.BinDir(), h.Binary),
		Markers: h.Markers,
	}
}

// Plan lists, in plain words, what Install would do.
func (e Env) Plan(it catalog.Item, s Scope) []string {
	var steps []string
	for _, b := range it.Binaries {
		steps = append(steps, fmt.Sprintf("build %s -> %s", b, filepath.Join(e.BinDir(), b)))
	}
	if it.Hook != nil {
		steps = append(steps, fmt.Sprintf("add %s hook to %s", it.Hook.Event, e.SettingsPath(s)))
	}
	return steps
}

// Install builds the binaries and adds the hook entry.
func (e Env) Install(it catalog.Item, s Scope) ([]string, error) {
	if e.RepoRoot == "" {
		return nil, errors.New("dev-tools source not found: run devtools from inside the repo")
	}
	if err := os.MkdirAll(e.BinDir(), 0o755); err != nil {
		return nil, err
	}
	var log []string
	for _, b := range it.Binaries {
		dest := filepath.Join(e.BinDir(), b)
		// go build refuses to overwrite a non-binary (e.g. a legacy script).
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return log, err
		}
		cmd := exec.Command("go", "build", "-o", dest, "./cmd/"+b)
		cmd.Dir = e.RepoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			return log, fmt.Errorf("go build %s: %v\n%s", b, err, out)
		}
		log = append(log, "built "+dest)
	}
	if it.Hook != nil {
		path := e.SettingsPath(s)
		changed, backup, err := AddHook(path, e.hookEntry(it.Hook))
		if err != nil {
			return log, err
		}
		if changed {
			log = append(log, "hook added to "+path)
			if backup != "" {
				log = append(log, "backup: "+backup)
			}
		} else {
			log = append(log, "hook already present in "+path)
		}
	}
	return log, nil
}

// Uninstall removes the hook entry and the binaries.
func (e Env) Uninstall(it catalog.Item, s Scope) ([]string, error) {
	var log []string
	if it.Hook != nil {
		path := e.SettingsPath(s)
		changed, backup, err := RemoveHook(path, e.hookEntry(it.Hook))
		if err != nil {
			return log, err
		}
		if changed {
			log = append(log, "hook removed from "+path)
			if backup != "" {
				log = append(log, "backup: "+backup)
			}
		}
	}
	for _, b := range it.Binaries {
		dest := filepath.Join(e.BinDir(), b)
		if err := os.Remove(dest); err == nil {
			log = append(log, "removed "+dest)
		}
	}
	if len(log) == 0 {
		log = append(log, "nothing to remove")
	}
	return log, nil
}

// Status describes the install state of an item.
func (e Env) Status(it catalog.Item, s Scope) string {
	var missing []string
	for _, b := range it.Binaries {
		if _, err := os.Stat(filepath.Join(e.BinDir(), b)); err != nil {
			missing = append(missing, b)
		}
	}
	hook := true
	if it.Hook != nil {
		hook = HasHook(e.SettingsPath(s), e.hookEntry(it.Hook))
	}
	switch {
	case len(missing) == 0 && hook:
		return "installed"
	case len(missing) == len(it.Binaries) && !hook, len(missing) == len(it.Binaries) && it.Hook == nil:
		return "not installed"
	default:
		return "partial"
	}
}

// PathWarning returns a hint when the bin dir is not on PATH.
func (e Env) PathWarning() string {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == e.BinDir() {
			return ""
		}
	}
	return fmt.Sprintf("%s is not on your PATH; add: export PATH=\"$HOME/.local/bin:$PATH\"", e.BinDir())
}
