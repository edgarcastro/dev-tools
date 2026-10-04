// claude-agent automates tmux + git worktrees to run several Claude Code
// agents in parallel, each in its own branch, worktree and tmux session.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/edgarcastro/dev-tools/internal/agent"
	"github.com/edgarcastro/dev-tools/internal/tui"
)

const usage = `Usage: claude-agent <command>

  new [task] [base-branch]   Create worktree + branch + tmux session and launch Claude Code
  attach                     Pick among active agents and attach
  list                       List active agents
  cleanup                    Close a session and optionally delete its worktree/branch
`

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	var err error
	switch cmd {
	case "new":
		err = cmdNew(os.Args[2:])
	case "attach":
		err = cmdAttach()
	case "list":
		cmdList()
	case "cleanup":
		err = cmdCleanup()
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
	if err != nil {
		if errors.Is(err, tui.ErrCancelled) {
			fmt.Println("Cancelled.")
			return
		}
		fmt.Fprintln(os.Stderr, tui.Fatal(err))
		os.Exit(1)
	}
}

func cmdNew(args []string) error {
	var task, base string
	if len(args) > 0 {
		task = args[0]
	}
	if len(args) > 1 {
		base = args[1]
	}
	if task == "" {
		var err error
		if task, err = tui.Input("Task name", "fix-login"); err != nil {
			return err
		}
		if task == "" {
			return tui.ErrCancelled
		}
	}
	if base == "" {
		base = agent.DefaultBase()
	}
	session, err := agent.Create(task, base)
	if err != nil {
		return err
	}
	return agent.Enter(session)
}

func pickSession(title string) (agent.Session, error) {
	sessions := agent.Sessions()
	if len(sessions) == 0 {
		return agent.Session{}, errors.New("no active agents")
	}
	items := make([]tui.Item, len(sessions))
	byName := map[string]agent.Session{}
	for i, s := range sessions {
		items[i] = tui.Item{Value: s.Name, Label: s.Task, Desc: fmt.Sprintf("%s window(s), created %s", s.Windows, s.Created)}
		byName[s.Name] = s
	}
	name, err := tui.Pick(title, items)
	if err != nil {
		return agent.Session{}, err
	}
	return byName[name], nil
}

func cmdAttach() error {
	s, err := pickSession("Attach to agent")
	if err != nil {
		return err
	}
	return agent.Enter(s.Name)
}

func cmdList() {
	sessions := agent.Sessions()
	if len(sessions) == 0 {
		fmt.Println("No active agents.")
		return
	}
	for _, s := range sessions {
		fmt.Printf("%s: %s window(s), created %s\n", s.Name, s.Windows, s.Created)
	}
}

func cmdCleanup() error {
	s, err := pickSession("Close agent")
	if err != nil {
		return err
	}
	remove, err := tui.Confirm(fmt.Sprintf("Also delete the worktree and branch '%s'?", s.Task), false)
	if err != nil {
		return err
	}
	if err := agent.Kill(s.Task, remove); err != nil {
		return err
	}
	fmt.Println(tui.OK.Render("Closed " + s.Name))
	return nil
}
