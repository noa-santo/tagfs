package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/noa-santo/tagfs/internal/fuse"
)

type focusArea int

const (
	focusList focusArea = iota
	focusTagInput
	focusApply
	focusCancel
	focusCount
)

type model struct {
	socketPath  string
	items       []fuse.InboxEntry
	cursor      int
	focus       focusArea
	err         error
	loading     bool
	width       int
	height      int
	spinner     spinner.Model
	tagInput    textinput.Model
	pendingTags map[string][]string
	keys        keyMap
	help        help.Model
}

func initialModel(socketPath string) model {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(colorMauve)),
	)

	ti := textinput.New()
	ti.Placeholder = "type a tag and press enter"
	ti.CharLimit = 48
	ti.SetWidth(32)

	h := help.New()
	h.Styles = help.DefaultStyles(true)

	return model{
		socketPath:  socketPath,
		items:       []fuse.InboxEntry{},
		loading:     true,
		focus:       focusList,
		spinner:     sp,
		tagInput:    ti,
		pendingTags: make(map[string][]string),
		keys:        defaultKeyMap(),
		help:        h,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchInboxItems(m.socketPath), m.spinner.Tick)
}

func (m model) selectedItem() (fuse.InboxEntry, bool) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return fuse.InboxEntry{}, false
	}
	return m.items[m.cursor], true
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)
		return m, nil

	case []fuse.InboxEntry:
		m.items = msg
		m.loading = false
		return m, nil

	case error:
		m.err = msg
		m.loading = false
		return m, nil

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if m.focus == focusTagInput && m.tagInput.Focused() {
			switch msg.String() {
			case "esc":
				m.tagInput.Blur()
				m.focus = focusList
				return m, nil
			case "enter":
				if item, ok := m.selectedItem(); ok {
					if tag := strings.TrimSpace(m.tagInput.Value()); tag != "" {
						m.pendingTags[item.Name] = append(m.pendingTags[item.Name], tag)
						m.tagInput.SetValue("")
					}
				}
				return m, nil
			case "tab":
				m.tagInput.Blur()
				m.focus = focusApply
				return m, nil
			}
			var cmd tea.Cmd
			m.tagInput, cmd = m.tagInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "q":
			return m, tea.Quit
		case "?":
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case "up", "k":
			if m.focus == focusList && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.focus == focusList && m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case "a":
			m.focus = focusTagInput
			return m, m.tagInput.Focus()

		case "tab":
			m.focus = (m.focus + 1) % focusCount
			if m.focus == focusTagInput {
				return m, m.tagInput.Focus()
			}
		case "shift+tab":
			if m.focus == focusList {
				m.focus = focusCancel
			} else {
				m.focus--
			}
			if m.focus == focusTagInput {
				return m, m.tagInput.Focus()
			}

		case "enter":
			switch m.focus {
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

func (m model) headerView() string {
	left := titleStyle.Render(iconInbox+" TAGFS") + "  " + subtitleStyle.Render("INBOX MANAGER")
	right := countStyle.Render(fmt.Sprintf("%d ITEMS", len(m.items)))

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	div := dividerStyle.Render(strings.Repeat("─", max(m.width, 0)))
	return lipgloss.JoinVertical(lipgloss.Left, line, div)
}

func (m model) listPanel(width, height int) string {
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render(iconFolder+" INBOX") + "\n\n")

	if len(m.items) == 0 {
		b.WriteString(emptyHintStyle.Render("your home directory is tidy :3 nothing waiting to be tagged"))
	} else {
		for i, item := range m.items {
			icon := iconFile
			nameStyle := rowFileStyle
			if item.IsDir {
				icon = iconFolder
				nameStyle = rowDirStyle
			}

			cursor := "  "
			line := fmt.Sprintf("%s %s", icon, item.Name)

			if tags := m.pendingTags[item.Name]; len(tags) > 0 {
				line += rowDimStyle.Render(fmt.Sprintf("  %s%d", iconTag, len(tags)))
			}

			if i == m.cursor {
				cursor = iconChevron + " "
				line = rowSelectedStyle.Render(fmt.Sprintf(" %s %s ", icon, item.Name))
				if tags := m.pendingTags[item.Name]; len(tags) > 0 {
					line += " " + rowDimStyle.Render(fmt.Sprintf("%s%d", iconTag, len(tags)))
				}
			} else {
				line = nameStyle.Render(line)
			}

			b.WriteString(cursor + line + "\n")
		}
	}

	style := panelStyle
	if m.focus == focusList {
		style = focusedPanelStyle
	}
	return style.Width(width).Height(height).Render(b.String())
}

func (m model) detailPanel(width, height int) string {
	var b strings.Builder

	item, ok := m.selectedItem()
	if !ok {
		b.WriteString(emptyHintStyle.Render("select an item to inspect and tag it"))
		return panelStyle.Width(width).Height(height).Render(b.String())
	}

	icon := iconFile
	kind := "FILE"
	if item.IsDir {
		icon = iconFolder
		kind = "DIRECTORY"
	}

	var info = kind
	if item.ModifiedAt != "" {
		info += " | Last modified: " + item.ModifiedAt
	}

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorText).Render(icon+"  "+item.Name) + "\n")
	b.WriteString(rowDimStyle.Render(info) + "\n\n")

	b.WriteString(sectionLabelStyle.Render(iconTags+" PENDING TAGS") + "\n")
	if tags := m.pendingTags[item.Name]; len(tags) > 0 {
		chips := make([]string, len(tags))
		for i, t := range tags {
			chips[i] = tagChipStyle.Render(iconTag + " " + t)
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, chips...) + "\n")
	} else {
		b.WriteString(emptyHintStyle.Render("no tags yet — add one below") + "\n")
	}

	b.WriteString("\n" + sectionLabelStyle.Render(iconMagic+" SUGGESTED TAGS") + "\n")
	b.WriteString(emptyHintStyle.Render("waiting on the daemon to learn your tagging habits") + "\n\n")

	inputStyle := inputBoxStyle
	if m.focus == focusTagInput {
		inputStyle = inputBoxFocusedStyle
	}
	b.WriteString(sectionLabelStyle.Render(iconPlus+" ADD TAG") + "\n")
	b.WriteString(inputStyle.Width(width - 6).Render(m.tagInput.View()))

	style := panelStyle
	if m.focus == focusTagInput {
		style = focusedPanelStyle
	}
	return style.Width(width).Height(height).Render(b.String())
}

func (m model) footerView() string {
	div := dividerStyle.Render(strings.Repeat("─", max(m.width, 0)))

	addBtn := buttonStyle.Render(iconPlus + " Add Tag")
	applyBtn := buttonStyle.Foreground(colorGreen).Render(iconCheck + " Apply")
	cancelBtn := buttonStyle.Foreground(colorRed).MarginRight(0).Render(iconCross + " Cancel")

	switch m.focus {
	case focusTagInput:
		addBtn = buttonStyle.Background(colorPeach).Foreground(colorBase).Bold(true).Render(iconPlus + " Add Tag")
	case focusApply:
		applyBtn = buttonStyle.Background(colorGreen).Foreground(colorBase).Bold(true).Render(iconCheck + " Apply")
	case focusCancel:
		cancelBtn = buttonStyle.MarginRight(0).Background(colorRed).Foreground(colorBase).Bold(true).Render(iconCross + " Cancel")
	default:
		break
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, addBtn, applyBtn, cancelBtn)
	helpLine := m.help.View(m.keys)

	return lipgloss.JoinVertical(lipgloss.Left, div, buttons, helpLine)
}

func (m model) View() tea.View {
	if m.width == 0 {
		v := tea.NewView("starting tagfs…")
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	header := m.headerView()

	if m.loading {
		body := lipgloss.Place(m.width, m.height-lipgloss.Height(header)-1,
			lipgloss.Center, lipgloss.Center,
			m.spinner.View()+" loading inbox from daemon…")
		v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, body))
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if m.err != nil {
		body := lipgloss.Place(m.width, m.height-lipgloss.Height(header)-1,
			lipgloss.Center, lipgloss.Center,
			errStyle.Render(iconWarning+" could not reach tagfs daemon: "+m.err.Error()))
		v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, body))
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	footer := m.footerView()
	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer) - 1
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	listWidth := m.width * 4 / 10
	detailWidth := m.width - listWidth - 1
	panelHeight := bodyHeight - 2

	list := m.listPanel(listWidth-4, panelHeight)
	detail := m.detailPanel(detailWidth-4, panelHeight)

	body := lipgloss.JoinHorizontal(lipgloss.Top, list, " ", detail)

	view := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	v := tea.NewView(view)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
