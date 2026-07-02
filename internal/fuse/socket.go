package fuse

import (
	"net"
	"os"
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
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			logger.Fatalf("Error while closing socket: %v", err)
		}
	}(conn)
	// todo: handle connection
}
