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
	"github.com/noa-santo/tagfs/internal/logic"
	. "github.com/noa-santo/tagfs/internal/shared"
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

		var items []InboxEntry
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

func getImplicitTags(socketPath string, tags []string) ([]string, error) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error connecting to socket: %v", err))
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("Error closing connection:", err)
		}
	}(conn)

	tagsString, err := json.Marshal(tags)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error marshalling existing tags: %v", err))
	}
	_, err = conn.Write([]byte(fmt.Sprintf("GET_IMPLICIT_TAGS\n%s\n", tagsString)))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error writing to socket: %v", err))
	}
	var implicitTags []string
	if err := json.NewDecoder(conn).Decode(&implicitTags); err != nil {
		return nil, errors.New(fmt.Sprintf("Error reading from socket: %v", err))
	}
	return implicitTags, nil
}

func getSuggestions(socketPath string, filename string) (logic.TagSuggestion, error) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return logic.TagSuggestion{}, errors.New(fmt.Sprintf("Error connecting to socket: %v", err))
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("Error closing connection:", err)
		}
	}(conn)

	_, err = conn.Write([]byte(fmt.Sprintf("GET_SUGGESTIONS\n%s\n", filename)))
	if err != nil {
		return logic.TagSuggestion{}, errors.New(fmt.Sprintf("Error writing to socket: %v", err))
	}
	var suggestions logic.TagSuggestion
	if err := json.NewDecoder(conn).Decode(&suggestions); err != nil {
		return logic.TagSuggestion{}, errors.New(fmt.Sprintf("Error reading from socket: %v", err))
	}
	var options [][]string
	for _, option := range suggestions.Options {
		if option != nil {
			options = append(options, option)
		}
	}
	suggestions.Options = options
	return suggestions, nil
}

func getTargetDestination(socketPath string, tags []string) (logic.EffectiveDir, bool) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return logic.EffectiveDir{}, false
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("Error closing connection:", err)
		}
	}(conn)

	tagsString, err := json.Marshal(tags)
	if err != nil {
		return logic.EffectiveDir{}, false
	}
	_, err = conn.Write([]byte(fmt.Sprintf("GET_TARGET_DESTINATION\n%s\n", tagsString)))
	if err != nil {
		return logic.EffectiveDir{}, false
	}
	var resp fuse.TargetResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return logic.EffectiveDir{}, false
	}

	return resp.Target, resp.Unambiguous
}
