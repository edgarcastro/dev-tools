# dev-tools

Personal toolbox (Go + Bubble Tea) of scripts, Claude Code hooks and skills, with an installer TUI. Module: `github.com/edgarcastro/dev-tools`. UI strings and docs are in English.

## Commands

- `make test`: `go vet ./...` + `go test ./...`
- `make build`: builds `bin/devtools` (gitignored)
- `make install`: builds and launches the interactive installer (needs a TTY)
- Non-interactive: `bin/devtools install|uninstall <id> [--project]`, `bin/devtools status`
- Go is at `/opt/homebrew/bin`; add it to `PATH` if `go` is not found.

## Layout

- `cmd/devtools`: installer TUI; `cmd/claude-agent`: worktree + tmux agent manager; `cmd/block-env-hook`: the PreToolUse hook binary
- `internal/catalog`: registry of installable items (add new items here)
- `internal/installer`: `go build` into `~/.local/bin` + idempotent `settings.json` hook merge/removal (timestamped backup first)
- `internal/envguard`: `.env*` matching logic (port of the old `block-env.sh`)
- `internal/agent`: git worktree + tmux logic; `internal/tui`: shared Bubble Tea prompts
- `items/skills`: reserved for Claude Code skills

## Adding an item

1. `cmd/<name>/main.go`
2. Register in `internal/catalog/catalog.go` (binaries, optional hook with `Markers` so reinstall/uninstall find legacy entries)
3. Tests + docs (`docs/<name>.md`, README table)

## Gotchas

- The installed block-env hook inspects Bash command text, so any Bash command containing a `.env` path or word (heredocs, commit messages, test payloads) is blocked. Create such content with the Write tool, put hook test payloads in files and feed them with `< file`, and reword commit messages.
- tmux format output: do not use a tab as a field separator (tmux sanitizes it); use `|`. `#{session_created_string}` is unsupported; use `#{t:session_created}`.
- `go build` refuses to overwrite non-binary files, so the installer removes the destination first.
- tmux integration test uses a short `/tmp` socket dir (unix socket path length limit) and a fake `claude`; it never touches the real tmux server.
- Project-scope hook installs go to `.claude/settings.local.json` (absolute path to the home dir, must not be committed).
- Reinstall after changing a tool (`bin/devtools install <id>`), since the installed copies in `~/.local/bin` are builds, not symlinks.
