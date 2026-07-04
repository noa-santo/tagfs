package fuse

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func startCommandListener() {
	socketPath := os.Getenv("TAGFS_SOCKET")
	if socketPath == "" {
		logger.Panicf("Socket path was not initialized! Set it at build time!")
	}
	_, err := os.Stat(socketPath)
	if !os.IsNotExist(err) {
		err := os.Remove(socketPath)
		if err != nil {
			logger.Fatalf("Failed to remove old socket!")
		}
	}
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		logger.Fatalf("Failed to listen on socket: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Println("Signal received, cleaning up socket...")
		err := l.Close()
		if err != nil {
			logger.Fatalf("Failed to close socket: %v", err)
		}
		err = os.Remove(socketPath)
		if err != nil {
			logger.Fatalf("Failed to remove socket: %v", err)
		}
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

func handleListInbox(conn net.Conn) {
	entries, err := getInboxEntries()
	if err != nil {
		inboxLogger.Printf("Error pulling entries: %v", err)
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(entries)
	err = json.NewEncoder(conn).Encode(entries)
	if err != nil {
		inboxLogger.Printf("Error writing entries: %v", err)
	}
}

func handleConnection(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			logger.Fatalf("Error while closing socket: %v", err)
		}
	}(conn)

	reader := bufio.NewReader(conn)
	cmd, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	switch cmd {
	case "PING\n":
		_, _ = conn.Write([]byte("PONG\n"))
		break
	case "LIST_INBOX\n":
		handleListInbox(conn)
		break
	default:
		logger.Printf("Unknown command: %s", cmd)
	}
}
