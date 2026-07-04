package fuse

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/noa-santo/tagfs/internal/config"
)

func removeIfExist(path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return
	}
	err = os.Remove(path)
	if err != nil {
		logger.Fatalf("Failed to remove socket: %v", err)
	}
}

func startCommandListener() {
	socketPath := os.Getenv("TAGFS_SOCKET")
	if socketPath == "" {
		logger.Panicf("Socket path was not initialized! Set it at build time!")
	}
	removeIfExist(socketPath)
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
		removeIfExist(socketPath)
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

func handleListTags(conn net.Conn) {
	tags := config.Get().GetAllTags()
	tags = append(tags, "overwrite")
	if err := json.NewEncoder(conn).Encode(tags); err != nil {
		logger.Printf("Error writing tags: %v", err)
	}
}

func handleCheckTagCompatibility(conn net.Conn) {
	reader := bufio.NewReader(conn)
	existingTagsString, err := reader.ReadString('\n')
	if err != nil {
		logger.Printf("Error reading existing tags: %v", err)
		return
	}
	existingTags := make([]string, 0)
	if err = json.Unmarshal([]byte(existingTagsString), &existingTags); err != nil {
		logger.Printf("Error unmarshalling existing tags: %v", err)
		return
	}

	newTag, err := reader.ReadString('\n')
	if err != nil {
		logger.Printf("Error reading new tag: %v", err)
		return
	}
	if err = json.NewEncoder(conn).Encode(config.Get().IsTagCompatible(newTag, existingTags)); err != nil {
		logger.Printf("Error writing compatibility: %v", err)
		return
	}
}

func handleGetImplicitTags(conn net.Conn) {
	reader := bufio.NewReader(conn)
	tagsString, err := reader.ReadString('\n')
	if err != nil {
		logger.Printf("Error reading existing tags: %v", err)
		return
	}
	tags := make([]string, 0)
	if err = json.Unmarshal([]byte(tagsString), &tags); err != nil {
		logger.Printf("Error unmarshalling existing tags: %v", err)
		return
	}
	if err = json.NewEncoder(conn).Encode(config.Get().GetImplicitTags(tags)); err != nil {
		logger.Printf("Error writing implicit tags: %v", err)
		return
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
	case "LIST_TAGS\n":
		handleListTags(conn)
		break
	case "CHECK_TAG_COMPATIBILITY\n":
		handleCheckTagCompatibility(conn)
		break
	case "GET_IMPLICIT_TAGS\n":
		handleGetImplicitTags(conn)
		break
	default:
		logger.Printf("Unknown command: %s", cmd)
	}
}
