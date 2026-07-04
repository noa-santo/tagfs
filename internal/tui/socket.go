package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/noa-santo/tagfs/internal/fuse"
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

func fetchInboxItems(socketPath string) tea.Cmd {
	return func() tea.Msg {
		conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
		if err != nil {
			return err
		}
		defer func(conn net.Conn) {
			err := conn.Close()
			if err != nil {
				fmt.Println("Error closing connection:", err)
			}
		}(conn)

		if _, err := conn.Write([]byte("LIST_INBOX\n")); err != nil {
			return err
		}

		var items []fuse.InboxEntry
		if err := json.NewDecoder(conn).Decode(&items); err != nil {
			return err
		}

		return items
	}
}

func fetchTags(socketPath string) tea.Cmd {
	return func() tea.Msg {
		conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
		if err != nil {
			return err
		}
		defer func(conn net.Conn) {
			err := conn.Close()
			if err != nil {
				fmt.Println("Error closing connection:", err)
			}
		}(conn)

		if _, err := conn.Write([]byte("LIST_TAGS\n")); err != nil {
			return err
		}

		var tags []string
		if err := json.NewDecoder(conn).Decode(&tags); err != nil {
			return err
		}

		return tagMsg(tags)
	}
}

func isTagCompatible(socketPath string, newTag string, existingTags []string) (bool, error) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Error connecting to socket: %v", err))
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("Error closing connection:", err)
		}
	}(conn)

	existingTagsString, err := json.Marshal(existingTags)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Error marshalling existing tags: %v", err))
	}
	_, err = conn.Write([]byte(fmt.Sprintf("CHECK_TAG_COMPATIBILITY\n%s\n%s\n", existingTagsString, newTag)))
	if err != nil {
		return false, errors.New(fmt.Sprintf("Error writing to socket: %v", err))
	}
	var isCompatible bool
	if err := json.NewDecoder(conn).Decode(&isCompatible); err != nil {
		return false, errors.New(fmt.Sprintf("Error reading from socket: %v", err))
	}
	return isCompatible, nil
}
