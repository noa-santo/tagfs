package tui

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/noa-santo/tagfs/internal/fuse"
)

type focusArea int

const (
	focusList focusArea = iota
	focusAddTag
	focusApply
	focusCancel
)

type model struct {
	socketPath string
	items      []fuse.InboxEntry
	cursor     int
	focus      focusArea
	err        error
	loading    bool
}

type itemsMsg []fuse.InboxEntry
type errMsg error

func initialModel(socketPath string) model {
	return model{
		socketPath: socketPath,
		items:      []fuse.InboxEntry{},
		loading:    true,
		focus:      focusList,
	}
}

func (m model) fetchInboxItems(socketPath string) tea.Cmd {
	return func() tea.Msg {
		conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
		if err != nil {
			fmt.Println("Error connecting to daemon:", err)
			return errMsg(err)
		}
		defer func(conn net.Conn) {
			err := conn.Close()
			if err != nil {
				fmt.Println("Error closing connection:", err)
			}
		}(conn)

		_, err = conn.Write([]byte("LIST_INBOX\n"))
		if err != nil {
			return errMsg(err)
		}

		var items itemsMsg
		err = json.NewDecoder(conn).Decode(&items)
		if err != nil {
			return errMsg(err)
		}

		return items
	}
}

func (m model) Init() tea.Cmd {
	return m.fetchInboxItems(m.socketPath)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case itemsMsg:
		m.items = msg
		m.loading = false
		return m, nil

	case errMsg:
		m.err = msg
		m.loading = false
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.focus == focusList && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.focus == focusList && m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case "tab":
			m.focus = (m.focus + 1) % 4
		case "shift+tab":
			if m.focus == focusList {
				m.focus = focusCancel
			} else {
				m.focus--
			}

		case "enter":
			switch m.focus {
			case focusAddTag:
				// TODO: Add tags action
			case focusApply:
				// TODO: Apply action
			case focusCancel:
				return m, tea.Quit
			default:
				return m, nil
			}
		}

	case tea.MouseWheelMsg:
		if msg.Button == tea.MouseWheelUp {
			if m.focus == focusList && m.cursor > 0 {
				m.cursor--
			}
		} else if msg.Button == tea.MouseWheelDown {
			if m.focus == focusList && m.cursor < len(m.items)-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	var s strings.Builder

	// Header
	s.WriteString(" 📥 TAGFS INBOX MANAGER\n")
	s.WriteString(" ─────────────────────────────────────────────────\n\n")

	if m.loading {
		s.WriteString("  Loading items from daemon socket...\n\n")
		v := tea.NewView(s.String())
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if m.err != nil {
		s.WriteString(fmt.Sprintf("  ❌ Error connecting to daemon: %v\n\n", m.err))
		v := tea.NewView(s.String())
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if len(m.items) == 0 {
		s.WriteString("  No items currently in your inbox.\n\n")
	} else {
		for i, item := range m.items {
			cursorStr := "  "
			if m.cursor == i && m.focus == focusList {
				cursorStr = " >"
			}

			icon := " "
			if item.IsDir {
				icon = " "
			}

			s.WriteString(fmt.Sprintf("%s %s %s\n", cursorStr, icon, item.Name))
		}
		s.WriteString("\n")
	}

	s.WriteString(" ─────────────────────────────────────────────────\n")

	addTagBtn := "[ Add Tag ]"
	applyBtn := "[ Apply ]"
	cancelBtn := "[ Cancel ]"

	if m.focus == focusAddTag {
		addTagBtn = "▶\033[1;36m[ Add Tag ]\033[0m◀"
	} else if m.focus == focusApply {
		applyBtn = "▶\033[1;32m[ Apply ]\033[0m◀"
	} else if m.focus == focusCancel {
		cancelBtn = "▶\033[1;31m[ Cancel ]\033[0m◀"
	}

	s.WriteString(fmt.Sprintf("  %s     %s   %s\n\n", addTagBtn, applyBtn, cancelBtn))
	s.WriteString(" 💡 Navigation: j/k or 󰜮/󰜱 to scroll • Tab to cycle focus • Mouse Wheel to scroll\n")

	v := tea.NewView(s.String())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
