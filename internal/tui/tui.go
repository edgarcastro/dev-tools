// Package tui holds small reusable Bubble Tea prompts: text input, filterable
// picker, confirm and multi-select.
package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrCancelled is returned when the user aborts a prompt (esc / ctrl+c).
var ErrCancelled = errors.New("cancelled")

var (
	Title    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	Dim      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	OK       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	Warn     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	Err      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	Selected = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
)

// ---- text input ----

type inputModel struct {
	label string
	in    textinput.Model
	done  bool
	quit  bool
}

func (m inputModel) Init() tea.Cmd { return textinput.Blink }

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "enter":
			m.done = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.in, cmd = m.in.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	if m.done || m.quit {
		return ""
	}
	return Title.Render(m.label) + "\n" + m.in.View() + "\n" + Dim.Render("enter confirm · esc cancel") + "\n"
}

// Input asks for a line of text.
func Input(label, placeholder string) (string, error) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 80
	res, err := tea.NewProgram(inputModel{label: label, in: ti}).Run()
	if err != nil {
		return "", err
	}
	m := res.(inputModel)
	if m.quit {
		return "", ErrCancelled
	}
	return strings.TrimSpace(m.in.Value()), nil
}

// ---- filterable picker ----

// Item is one row of Pick.
type Item struct {
	Value string // returned on selection
	Label string
	Desc  string
}

func (i Item) Title() string       { return i.Label }
func (i Item) Description() string { return i.Desc }
func (i Item) FilterValue() string { return i.Label + " " + i.Desc }

type pickModel struct {
	list   list.Model
	choice string
	quit   bool
}

func (m pickModel) Init() tea.Cmd { return nil }

func (m pickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-1)
	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "enter":
				if it, ok := m.list.SelectedItem().(Item); ok {
					m.choice = it.Value
				}
				return m, tea.Quit
			case "esc", "q", "ctrl+c":
				m.quit = true
				return m, tea.Quit
			}
		} else if msg.String() == "ctrl+c" {
			m.quit = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m pickModel) View() string { return m.list.View() }

// Pick shows a filterable list ('/' to filter) and returns the chosen Value.
func Pick(title string, items []Item) (string, error) {
	li := make([]list.Item, len(items))
	for i, it := range items {
		li[i] = it
	}
	l := list.New(li, list.NewDefaultDelegate(), 80, min(len(items)*3+6, 24))
	l.Title = title
	l.SetShowStatusBar(false)
	res, err := tea.NewProgram(pickModel{list: l}).Run()
	if err != nil {
		return "", err
	}
	m := res.(pickModel)
	if m.quit || m.choice == "" {
		return "", ErrCancelled
	}
	return m.choice, nil
}

// ---- confirm ----

type confirmModel struct {
	question string
	yes      bool
	answered bool
	def      bool
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch strings.ToLower(k.String()) {
		case "y":
			m.yes, m.answered = true, true
			return m, tea.Quit
		case "n", "esc", "ctrl+c":
			m.answered = true
			return m, tea.Quit
		case "enter":
			m.yes, m.answered = m.def, true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.answered {
		return ""
	}
	hint := "[y/N]"
	if m.def {
		hint = "[Y/n]"
	}
	return Warn.Render(m.question) + " " + Dim.Render(hint) + "\n"
}

// Confirm asks a yes/no question; def is the answer for a bare enter.
func Confirm(question string, def bool) (bool, error) {
	res, err := tea.NewProgram(confirmModel{question: question, def: def}).Run()
	if err != nil {
		return false, err
	}
	return res.(confirmModel).yes, nil
}

// Fatal prints an error in red to stderr-friendly text.
func Fatal(err error) string { return Err.Render(fmt.Sprintf("error: %v", err)) }
