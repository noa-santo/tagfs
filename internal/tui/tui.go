package tui

import (
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
)

func StartTui() {
	socketPath := os.Getenv("TAGFS_SOCKET")
	if socketPath == "" {
		log.Fatal("Environment variable TAGFS_SOCKET is not set!")
	}

	m := initialModel(socketPath)
	_, err := tea.NewProgram(m).Run()
	if err != nil {
		log.Panicf("Error while running TUI: %v", err)
	}
}
