package fuse

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/noa-santo/tagfs/internal/logic"
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
	tags := logic.GetAllTags()
	tags = append(tags, "overwrite")
	if err := json.NewEncoder(conn).Encode(tags); err != nil {
		logger.Printf("Error writing tags: %v", err)
	}
}

func handleCheckTagCompatibility(conn net.Conn, reader *bufio.Reader) {
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

	newTagRaw, err := reader.ReadString('\n')
	if err != nil {
		logger.Printf("Error reading new tag: %v", err)
		return
	}
	newTag := strings.TrimSpace(strings.TrimSuffix(newTagRaw, "\n"))
	if err = json.NewEncoder(conn).Encode(logic.IsTagCompatible(newTag, existingTags)); err != nil {
		logger.Printf("Error writing compatibility: %v", err)
		return
	}
}

func handleGetImplicitTags(conn net.Conn, reader *bufio.Reader) {
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
	if err = json.NewEncoder(conn).Encode(logic.GetImplicitTags(tags)); err != nil {
		logger.Printf("Error writing implicit tags: %v", err)
		return
	}
}

func handleGetSuggestions(conn net.Conn, reader *bufio.Reader) {
	fileNameString, err := reader.ReadString('\n')
	if err != nil {
		logger.Printf("Error reading file name: %v", err)
		return
	}
	fileName := strings.TrimSuffix(fileNameString, "\n")
	entry, err := getInboxEntry(fileName)
	if err != nil {
		logger.Printf("Error reading inbox entry: %v", err)
		return
	}
	tagSuggestion := logic.SuggestTags(entry)
	err = json.NewEncoder(conn).Encode(tagSuggestion)
	if err != nil {
		logger.Printf("Error writing suggestions: %v", err)
	}
}

type TargetResponse struct {
	Target      logic.EffectiveDir `json:"target"`
	Unambiguous bool               `json:"unambiguous"`
}

func handleGetTargetDestination(conn net.Conn, reader *bufio.Reader) {
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
	target, unambiguous := logic.GetTargetDestination(tags)
	if err := json.NewEncoder(conn).Encode(TargetResponse{
		Target:      target,
		Unambiguous: unambiguous,
	}); err != nil {
		logger.Printf("Error writing response: %v", err)
	}
}

func handleUpdateTags(conn net.Conn, reader *bufio.Reader) {
	tagMapString, err := reader.ReadString('\n')
	if err != nil {
		logger.Printf("Error reading existing tags: %v", err)
		conn.Write([]byte("ERROR" + err.Error() + "\n"))
		return
	}
	tagMap := make(map[string][]string)
	if err = json.Unmarshal([]byte(tagMapString), &tagMap); err != nil {
		logger.Printf("Error unmarshalling existing tags: %v", err)
		conn.Write([]byte("ERROR" + err.Error() + "\n"))
		return
	}
	logger.Printf("Updating tags: %v", tagMap)
	_, err = conn.Write([]byte("OK\n"))
	if err != nil {
		logger.Printf("Error writing response: %v", err)
		return
	}
	// todo: actually update tags
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
		handleCheckTagCompatibility(conn, reader)
		break
	case "GET_IMPLICIT_TAGS\n":
		handleGetImplicitTags(conn, reader)
		break
	case "GET_SUGGESTIONS\n":
		handleGetSuggestions(conn, reader)
		break
	case "GET_TARGET_DESTINATION\n":
		handleGetTargetDestination(conn, reader)
	case "UPDATE_TAGS\n":
		handleUpdateTags(conn, reader)
	default:
		logger.Printf("Unknown command: %s", cmd)
	}
}
