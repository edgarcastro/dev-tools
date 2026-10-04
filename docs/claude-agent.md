# claude-agent — tmux + Claude Code session automation

Runs several Claude Code agents in parallel. Each task gets its own git
branch, worktree (`<repo>-worktrees/<task>`) and tmux session (`claude-<task>`).

## Install

```bash
brew install tmux
make install            # select "claude-agent" in the installer
export PATH="$HOME/.local/bin:$PATH"   # add to ~/.zshrc if needed
```

## Commands

| Command | Description |
| --- | --- |
| `claude-agent new [task] [base-branch]` | Create worktree + branch + tmux session and launch Claude Code. Prompts for the task name if omitted; base defaults to `main` (or the current `HEAD` if there is no `main`) |
| `claude-agent attach` | Filterable picker (`/` to filter) among active agents |
| `claude-agent list` | List active agents |
| `claude-agent cleanup` | Close a session and optionally delete its worktree and branch |

Task names may contain letters, digits, `-` and `_` only.

## Ghostty shortcuts (`~/.config/ghostty/config`)

```
keybind = cmd+shift+n=text:claude-agent new\n
keybind = cmd+shift+j=text:claude-agent attach\n
keybind = cmd+shift+x=text:claude-agent cleanup\n
```
