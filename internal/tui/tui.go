package tui

import (
	"log"
	"net"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
)

func testSocket(socketPath string) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		log.Println("Error connecting to socket:", err)
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("Error closing connection: %v", err)
		}
	}(conn)

	if _, err := conn.Write([]byte("PING\n")); err != nil {
		log.Panicf("Error writing to socket: %v", err)
	}

	msg := make([]byte, 1024)
	n, err := conn.Read(msg)
	if err != nil {
		log.Panicf("Error reading from socket: %v", err)
	}
	msg = msg[:n]
	if string(msg) != "PONG\n" {
		log.Panicf("Unexpected response from socket: %s", string(msg))
	}
}

func StartTui() {
	socketPath := os.Getenv("TAGFS_SOCKET")
	if socketPath == "" {
		log.Fatal("Environment variable TAGFS_SOCKET is not set!")
	}
	testSocket(socketPath)

	m := initialModel(socketPath)
	_, err := tea.NewProgram(m).Run()
	if err != nil {
		log.Panicf("Error while running TUI: %v", err)
	}
}
