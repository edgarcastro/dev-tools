package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type multiModel struct {
	title   string
	items   []Item
	cursor  int
	checked map[int]bool
	done    bool
	quit    bool
}

func (m multiModel) Init() tea.Cmd { return nil }

func (m multiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "up", "k":
		m.cursor = (m.cursor + len(m.items) - 1) % len(m.items)
	case "down", "j":
		m.cursor = (m.cursor + 1) % len(m.items)
	case " ", "x":
		m.checked[m.cursor] = !m.checked[m.cursor]
	case "a":
		all := len(m.checked) != len(m.items)
		m.checked = map[int]bool{}
		for i := range m.items {
			if all {
				m.checked[i] = true
			}
		}
	case "enter":
		if len(m.checked) == 0 {
			m.checked[m.cursor] = true
		}
		m.done = true
		return m, tea.Quit
	case "esc", "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	}
	for i, v := range m.checked {
		if !v {
			delete(m.checked, i)
		}
	}
	return m, nil
}

func (m multiModel) View() string {
	if m.done || m.quit {
		return ""
	}
	var b strings.Builder
	b.WriteString(Title.Render(m.title) + "\n\n")
	for i, it := range m.items {
		cur, box := "  ", "[ ]"
		if m.checked[i] {
			box = "[x]"
		}
		label := it.Label
		if i == m.cursor {
			cur = "> "
			label = Selected.Render(label)
		}
		b.WriteString(cur + box + " " + label + "\n")
		if it.Desc != "" {
			b.WriteString("      " + Dim.Render(it.Desc) + "\n")
		}
	}
	b.WriteString("\n" + Dim.Render("space toggle · a all · enter confirm · esc cancel") + "\n")
	return b.String()
}

// MultiSelect returns the Values of the checked items (the item under the
// cursor if none are checked).
func MultiSelect(title string, items []Item) ([]string, error) {
	res, err := tea.NewProgram(multiModel{title: title, items: items, checked: map[int]bool{}}).Run()
	if err != nil {
		return nil, err
	}
	m := res.(multiModel)
	if m.quit {
		return nil, ErrCancelled
	}
	var out []string
	for i, it := range items {
		if m.checked[i] {
			out = append(out, it.Value)
		}
	}
	return out, nil
}
