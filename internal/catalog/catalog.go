// Package catalog is the registry of items the installer can install.
// To add a new tool: add a cmd/<name> main package and an Item here.
package catalog

// Hook describes a Claude Code hook entry that runs one of the item's binaries.
type Hook struct {
	Event   string   // e.g. PreToolUse
	Matcher string   // tool matcher
	Binary  string   // binary (from Item.Binaries) the hook runs
	Markers []string // substrings identifying our entry, incl. legacy scripts, for replace/uninstall
}

// Item is something installable.
type Item struct {
	ID       string
	Title    string
	Desc     string
	Binaries []string // cmd/<name> packages built into the bin dir
	Hook     *Hook    // optional settings.json hook entry
}

// Items returns every installable item.
func Items() []Item {
	return []Item{
		{
			ID:       "block-env",
			Title:    "block-env hook",
			Desc:     "Claude Code PreToolUse hook that blocks reading/editing .env* files (except .env.example)",
			Binaries: []string{"block-env-hook"},
			Hook: &Hook{
				Event:   "PreToolUse",
				Matcher: "Read|Edit|Write|MultiEdit|NotebookEdit|Grep|Glob|Bash",
				Binary:  "block-env-hook",
				Markers: []string{"block-env-hook", "block-env.sh"},
			},
		},
		{
			ID:       "claude-agent",
			Title:    "claude-agent",
			Desc:     "Run several Claude Code agents in parallel: git worktree + tmux session per task",
			Binaries: []string{"claude-agent"},
		},
	}
}

// Find returns the item with the given ID.
func Find(id string) (Item, bool) {
	for _, it := range Items() {
		if it.ID == id {
			return it, true
		}
	}
	return Item{}, false
}
