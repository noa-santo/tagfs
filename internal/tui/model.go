package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

type model struct {
	files  []string
	cursor int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
			}
		case "enter":
			// THIS IS WHERE YOU TRIGGER THE MOVE
			fmt.Printf("Sorting: %s\n", m.files[m.cursor])
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	s := "TagFS Inbox Manager (q to quit, enter to sort)\n\n"
	for i, file := range m.files {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, file)
	}
	return s
}
