package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Exercises Create/Sessions/Kill against a real git repo and a private tmux
// server, with a fake `claude` so nothing real is launched.
func TestCreateListKill(t *testing.T) {
	for _, bin := range []string{"git", "tmux"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not installed", bin)
		}
	}
	tmp := t.TempDir()
	fakeBin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fakeBin, "claude"), []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeBin+":"+os.Getenv("PATH"))
	sock, err := os.MkdirTemp("/tmp", "ca") // short path: unix sockets have a length limit
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(sock) })
	t.Setenv("TMUX_TMPDIR", sock) // isolated tmux server socket
	t.Setenv("TMUX", "")

	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "init"},
	} {
		if out, err := run(repo, "git", args...); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = run("", "tmux", "kill-server") })

	session, err := Create("it-task", "main")
	if err != nil {
		t.Fatal(err)
	}
	if session != "claude-it-task" {
		t.Fatalf("session = %q", session)
	}
	if _, err := os.Stat(filepath.Join(WorktreeBase(RepoRoot()), "it-task")); err != nil {
		t.Fatalf("worktree missing: %v", err)
	}
	if got := Sessions(); len(got) != 1 || got[0].Task != "it-task" {
		t.Fatalf("Sessions = %+v", got)
	}
	if _, err := Create("it-task", "main"); err != nil { // idempotent
		t.Fatalf("second Create: %v", err)
	}
	if err := Kill("it-task", true); err != nil {
		t.Fatal(err)
	}
	if got := Sessions(); len(got) != 0 {
		t.Fatalf("session not killed: %+v", got)
	}
	if out, _ := run(repo, "git", "branch", "--list", "it-task"); out != "" {
		t.Fatalf("branch not deleted: %q", out)
	}
}
