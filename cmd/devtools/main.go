// devtools installs the tools in this repo (binaries, Claude Code hooks).
//
//	devtools                          interactive installer
//	devtools install <id> [--project] non-interactive install
//	devtools uninstall <id> [--project]
//	devtools status
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/edgarcastro/dev-tools/internal/catalog"
	"github.com/edgarcastro/dev-tools/internal/installer"
	"github.com/edgarcastro/dev-tools/internal/tui"
)

func main() {
	env, err := installer.NewEnv()
	if err != nil {
		fail(err)
	}
	args := os.Args[1:]
	switch {
	case len(args) == 0:
		err = interactive(env)
	case args[0] == "status":
		status(env, installer.User)
	case (args[0] == "install" || args[0] == "uninstall") && len(args) >= 2:
		err = direct(env, args)
	default:
		fmt.Fprintln(os.Stderr, "Usage: devtools [install|uninstall <id> [--project] | status]")
		os.Exit(1)
	}
	if err != nil {
		if errors.Is(err, tui.ErrCancelled) {
			fmt.Println("Cancelled.")
			return
		}
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, tui.Fatal(err))
	os.Exit(1)
}

func status(env installer.Env, s installer.Scope) {
	for _, it := range catalog.Items() {
		fmt.Printf("%-14s %s\n", it.ID, env.Status(it, s))
	}
}

func direct(env installer.Env, args []string) error {
	scope := installer.User
	for _, a := range args[2:] {
		if a == "--project" {
			scope = installer.Project
		}
	}
	it, ok := catalog.Find(args[1])
	if !ok {
		return fmt.Errorf("unknown item %q", args[1])
	}
	return apply(env, args[0], []catalog.Item{it}, scope)
}

func interactive(env installer.Env) error {
	action, err := tui.Pick("dev-tools", []tui.Item{
		{Value: "install", Label: "Install", Desc: "Build and install tools"},
		{Value: "uninstall", Label: "Uninstall", Desc: "Remove installed tools"},
		{Value: "status", Label: "Status", Desc: "Show what is installed"},
	})
	if err != nil {
		return err
	}
	if action == "status" {
		status(env, installer.User)
		return nil
	}

	var rows []tui.Item
	for _, it := range catalog.Items() {
		rows = append(rows, tui.Item{Value: it.ID, Label: it.Title + "  (" + env.Status(it, installer.User) + ")", Desc: it.Desc})
	}
	ids, err := tui.MultiSelect("Select items to "+action, rows)
	if err != nil {
		return err
	}
	var items []catalog.Item
	needScope := false
	for _, id := range ids {
		it, _ := catalog.Find(id)
		items = append(items, it)
		needScope = needScope || it.Hook != nil
	}

	scope := installer.User
	if needScope {
		s, err := tui.Pick("Hook scope", []tui.Item{
			{Value: string(installer.User), Label: "User", Desc: env.SettingsPath(installer.User)},
			{Value: string(installer.Project), Label: "Project", Desc: env.SettingsPath(installer.Project)},
		})
		if err != nil {
			return err
		}
		scope = installer.Scope(s)
	}

	fmt.Println(tui.Title.Render("Plan"))
	for _, it := range items {
		if action == "install" {
			for _, step := range env.Plan(it, scope) {
				fmt.Println("  • " + step)
			}
		} else {
			fmt.Println("  • remove " + it.Title)
		}
	}
	if ok, err := tui.Confirm("Proceed?", true); err != nil || !ok {
		return tui.ErrCancelled
	}
	return apply(env, action, items, scope)
}

func apply(env installer.Env, action string, items []catalog.Item, scope installer.Scope) error {
	var failed bool
	for _, it := range items {
		var log []string
		var err error
		if action == "install" {
			log, err = env.Install(it, scope)
		} else {
			log, err = env.Uninstall(it, scope)
		}
		fmt.Println(tui.Title.Render(it.Title))
		for _, l := range log {
			fmt.Println("  " + tui.OK.Render("✓ ") + l)
		}
		if err != nil {
			failed = true
			fmt.Println("  " + tui.Err.Render("✗ "+strings.TrimSpace(err.Error())))
		}
	}
	if action == "install" {
		if w := env.PathWarning(); w != "" {
			fmt.Println(tui.Warn.Render("! " + w))
		}
		for _, it := range items {
			if it.ID == "claude-agent" {
				fmt.Println(tui.Dim.Render("  claude-agent needs tmux: brew install tmux"))
			}
		}
	}
	if failed {
		return errors.New("some steps failed")
	}
	return nil
}
