package tui

import (
	"log"

	tea "charm.land/bubbletea/v2"
)

func StartTui() {
	m := model{}
	_, err := tea.NewProgram(m).Run()
	if err != nil {
		log.Panicf("Error while running TUI: %v", err)
	}
}
