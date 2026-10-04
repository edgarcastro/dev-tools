// Package agent manages Claude Code agents: one git worktree + branch + tmux
// session per task.
package agent

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// Prefix is prepended to every tmux session name managed by claude-agent.
const Prefix = "claude-"

var taskRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// ValidateTask rejects names that would break branch or tmux session names
// (spaces, slashes, dots and colons).
func ValidateTask(task string) error {
	if !taskRe.MatchString(task) {
		return fmt.Errorf("invalid task name %q: use letters, digits, '-' and '_' only", task)
	}
	return nil
}

// Session is an active agent tmux session.
type Session struct {
	Name    string // full tmux session name, e.g. claude-fix-login
	Task    string // Name without the prefix
	Windows string
	Created string
}

func run(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		if s != "" {
			return s, fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), s)
		}
		return s, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return s, nil
}

// RepoRoot returns the main worktree of the current repository (so it also
// works when invoked from inside an agent worktree), or the cwd outside git.
func RepoRoot() string {
	if out, err := run("", "git", "worktree", "list", "--porcelain"); err == nil {
		for _, line := range strings.Split(out, "\n") {
			if p, ok := strings.CutPrefix(line, "worktree "); ok {
				return p
			}
		}
	}
	wd, _ := os.Getwd()
	return wd
}

// WorktreeBase is where task worktrees live: <repo>-worktrees.
func WorktreeBase(repoRoot string) string { return repoRoot + "-worktrees" }

// DefaultBase is "main" when it exists, otherwise the current HEAD.
func DefaultBase() string {
	if _, err := run("", "git", "rev-parse", "--verify", "--quiet", "main"); err == nil {
		return "main"
	}
	return "HEAD"
}

// Sessions lists active agent sessions.
func Sessions() []Session {
	out, err := run("", "tmux", "list-sessions", "-F",
		"#{session_name}|#{session_windows}|#{t:session_created}")
	if err != nil {
		return nil
	}
	var res []Session
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, "|")
		if len(f) != 3 || !strings.HasPrefix(f[0], Prefix) {
			continue
		}
		res = append(res, Session{Name: f[0], Task: strings.TrimPrefix(f[0], Prefix), Windows: f[1], Created: f[2]})
	}
	return res
}

// Create makes the worktree + branch (if missing) and the tmux session
// (if missing) running `claude`. It returns the session name.
func Create(task, base string) (string, error) {
	if err := ValidateTask(task); err != nil {
		return "", err
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		return "", errors.New("tmux not found: brew install tmux")
	}
	root := RepoRoot()
	dir := filepath.Join(WorktreeBase(root), task)
	session := Prefix + task

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(WorktreeBase(root), 0o755); err != nil {
			return "", err
		}
		if _, err := run(root, "git", "worktree", "add", "-b", task, dir, base); err != nil {
			return "", err
		}
	}
	if _, err := run("", "tmux", "has-session", "-t", "="+session); err != nil {
		if _, err := run("", "tmux", "new-session", "-d", "-s", session, "-c", dir, "claude"); err != nil {
			return "", err
		}
	}
	return session, nil
}

// Enter switches to the session when inside tmux, otherwise replaces this
// process with `tmux attach`.
func Enter(session string) error {
	target := "=" + session
	if os.Getenv("TMUX") != "" {
		_, err := run("", "tmux", "switch-client", "-t", target)
		return err
	}
	path, err := exec.LookPath("tmux")
	if err != nil {
		return err
	}
	return syscall.Exec(path, []string{"tmux", "attach", "-t", target}, os.Environ())
}

// Kill closes the session and optionally removes its worktree and branch.
func Kill(task string, removeWorktree bool) error {
	_, _ = run("", "tmux", "kill-session", "-t", "="+Prefix+task)
	if !removeWorktree {
		return nil
	}
	root := RepoRoot()
	var errs []error
	if _, err := run(root, "git", "worktree", "remove", filepath.Join(WorktreeBase(root), task), "--force"); err != nil {
		errs = append(errs, err)
	}
	if _, err := run(root, "git", "branch", "-D", task); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
