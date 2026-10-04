# dev-tools

Personal toolbox of scripts, Claude Code hooks and skills, written in Go with
[Bubble Tea](https://github.com/charmbracelet/bubbletea), plus an installer TUI
that puts each item in the right place.

## Items

| ID | What it is | Docs |
| --- | --- | --- |
| `block-env` | Claude Code `PreToolUse` hook that blocks reading/editing `.env*` files (except `.env.example`) | below |
| `claude-agent` | Run several Claude Code agents in parallel: git worktree + tmux session per task | [docs/claude-agent.md](docs/claude-agent.md) |

## Install

Requires Go (`brew install go`).

```bash
make install          # builds bin/devtools and launches the installer
```

The installer lets you install, uninstall or check status, choose items, and
pick the hook scope (user `~/.claude/settings.json`, or project
`.claude/settings.local.json`). Binaries go to `~/.local/bin`; settings files
are backed up (`*.bak-<timestamp>`) before every change.

Non-interactive:

```bash
bin/devtools install block-env [--project]
bin/devtools uninstall block-env
bin/devtools status
```

Make sure `~/.local/bin` is on your `PATH`.

## block-env

Blocks `Read`, `Edit`, `Write`, `MultiEdit`, `NotebookEdit`, `Grep`, `Glob` and
`Bash` calls that reference `.env`, `.env.local`, `--env-file=.env`,
`cat .env*`, etc. `.env.example` (as a whole path segment) is allowed. The hook
exits with code 2 and tells Claude why. Installing it replaces a legacy
`block-env.sh` entry in `settings.json` if present.

## Layout

```
cmd/devtools        installer TUI
cmd/claude-agent    agent manager
cmd/block-env-hook  the hook binary
internal/catalog    registry of installable items
internal/installer  build + settings.json merge
internal/envguard   .env matching logic
internal/agent      worktree + tmux logic
internal/tui        shared Bubble Tea prompts
items/skills        reserved for Claude Code skills
```

## Adding an item

1. Add `cmd/<name>/main.go`.
2. Register it in `internal/catalog/catalog.go` (binaries, optional hook).
3. Add tests and docs.

## Development

```bash
make test
```
