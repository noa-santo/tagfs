package tui

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	. "github.com/noa-santo/tagfs/internal/shared"
)

type tagMsg []string

type clearToastMsg int

type focusArea int

const (
	focusList focusArea = iota
	focusSuggested
	focusTagInput
	focusApply
	focusCancel
	focusCount
)

type selectableTag struct {
	name     string
	selected bool
}

type suggestionState struct {
	common      []selectableTag
	options     [][]selectableTag
	activeOpt   int
	cursorGroup int
	cursorIdx   int
}

type model struct {
	socketPath          string
	items               []InboxEntry
	cursor              int
	focus               focusArea
	err                 error
	loading             bool
	width               int
	height              int
	spinner             spinner.Model
	tagInput            textinput.Model
	pendingTags         map[string][]string
	implicitPendingTags map[string][]string
	suggestions         map[string]*suggestionState
	keys                keyMap
	help                help.Model
	toastMsg            string
	toastIsErr          bool
	toastID             int
}

func initialModel(socketPath string) model {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(colorMauve)),
	)

	ti := textinput.New()
	ti.Placeholder = "type a tag and press enter (delete to remove)"
	ti.CharLimit = 48
	ti.ShowSuggestions = true
	ti.SetWidth(48)

	h := help.New()
	h.Styles = help.DefaultStyles(true)

	return model{
		socketPath:          socketPath,
		items:               []InboxEntry{},
		loading:             true,
		focus:               focusList,
		spinner:             sp,
		tagInput:            ti,
		pendingTags:         make(map[string][]string),
		implicitPendingTags: make(map[string][]string),
		suggestions:         make(map[string]*suggestionState),
		keys:                defaultKeyMap(),
		help:                h,
	}
}

func (m model) setSuggestionState() (model, tea.Cmd) {
	currentSuggestion, err := getSuggestions(m.socketPath, m.items[m.cursor].Name)
	if err != nil {
		return m.showToast(fmt.Sprintf("could not get suggestions: %s", err.Error()), true)
	}

	state := &suggestionState{
		activeOpt:   -1,
		cursorGroup: -1,
	}

	for _, t := range currentSuggestion.CommonTags {
		if slices.Contains(m.pendingTags[m.items[m.cursor].Name], t) {
			continue
		}
		state.common = append(state.common, selectableTag{name: t, selected: true})
	}

	for _, optGroup := range currentSuggestion.Options {
		var opt []selectableTag
		for _, t := range optGroup {
			opt = append(opt, selectableTag{name: t, selected: false})
		}
		state.options = append(state.options, opt)
	}

	if len(state.common) == 0 && len(state.options) > 0 {
		state.cursorGroup = 0
	}

	m.suggestions[m.items[m.cursor].Name] = state

	return m, nil
}

func (m model) showToast(msg string, isErr bool) (model, tea.Cmd) {
	m.toastMsg = msg
	m.toastIsErr = isErr
	m.toastID++

	currentID := m.toastID
	cmd := func() tea.Msg {
		time.Sleep(3 * time.Second)
		return clearToastMsg(currentID)
	}

	return m, cmd
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchInboxItems(m.socketPath), fetchTags(m.socketPath), m.spinner.Tick)
}

func (m model) selectedItem() (InboxEntry, bool) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return InboxEntry{}, false
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

	case clearToastMsg:
		if int(msg) == m.toastID {
			m.toastMsg = ""
		}
		return m, nil

	case []InboxEntry:
		m.items = msg
		m.loading = false
		m.setSuggestionState()
		return m, nil

	case tagMsg:
		m.tagInput.SetSuggestions(msg)
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
					tag := strings.TrimSpace(m.tagInput.CurrentSuggestion())
					if tag == "" && m.tagInput.Value() != "" {
						return m.showToast("tag has to be valid", true)
					}
					if tag == "" {
						return m.showToast("tag cannot be empty", true)
					}
					if slices.Contains(m.pendingTags[item.Name], tag) {
						return m.showToast(fmt.Sprintf("Tag '%s' already exists", tag), true)
					}
					if len(m.pendingTags[item.Name]) > 0 {
						compatible, err := isTagCompatible(m.socketPath, tag, m.pendingTags[item.Name])
						if err != nil {
							return m.showToast(fmt.Sprintf("could not check compatibility: %s", err.Error()), true)
						}
						if !compatible {
							return m.showToast(fmt.Sprintf("tag '%s' is not compatible with existing tags", tag), true)
						}
					}
					m.pendingTags[item.Name] = append(m.pendingTags[item.Name], tag)
					m.tagInput.SetValue("")
					sort.Strings(m.pendingTags[item.Name])
					var err error
					m.implicitPendingTags[item.Name], err = getImplicitTags(m.socketPath, m.pendingTags[item.Name])
					if err != nil {
						return m.showToast(fmt.Sprintf("could not get implicit tags: %s", err.Error()), true)
					}
				}

			case "delete":
				if m.tagInput.CurrentSuggestion() != "" {
					if item, ok := m.selectedItem(); ok {
						if !slices.Contains(m.pendingTags[item.Name], m.tagInput.CurrentSuggestion()) {
							return m.showToast("tag is not pending so it cannot be removed", true)
						}
						m.pendingTags[item.Name] = remove(m.pendingTags[item.Name], m.tagInput.CurrentSuggestion())
						var err error
						m.implicitPendingTags[item.Name], err = getImplicitTags(m.socketPath, m.pendingTags[item.Name])
						if err != nil {
							return m.showToast(fmt.Sprintf("could not get implicit tags: %s", err.Error()), true)
						}
					}
				}

			case "tab":
				m.tagInput.Blur()
				m.focus = focusApply
				return m, nil
			case "shift+tab":
				m.tagInput.Blur()
				m.focus = focusList
				return m, nil
			}
			var cmd tea.Cmd
			m.tagInput, cmd = m.tagInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		case "?":
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case "up", "k":
			if m.focus == focusList && m.cursor > 0 {
				m.cursor--
				m.setSuggestionState()
			}
		case "down", "j":
			if m.focus == focusList && m.cursor < len(m.items)-1 {
				m.cursor++
				m.setSuggestionState()
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
				break
			}
		}

		if m.focus == focusSuggested {
			item, ok := m.selectedItem()
			if !ok {
				return m, nil
			}

			state := m.suggestions[item.Name]
			if state == nil {
				return m, nil
			}

			switch msg.String() {
			case "up", "k":
				state.cursorGroup--
				if state.cursorGroup < -1 {
					state.cursorGroup = len(state.options) - 1
				}
				state.cursorIdx = 0

			case "down", "j":
				state.cursorGroup++
				if state.cursorGroup >= len(state.options) {
					state.cursorGroup = -1
				}
				state.cursorIdx = 0

			case "left", "h":
				state.cursorIdx--
				if state.cursorIdx < 0 {
					state.cursorIdx = 0
				}

			case "right", "l":
				state.cursorIdx++
				maxIdx := 0
				if state.cursorGroup == -1 && len(state.common) > 0 {
					maxIdx = len(state.common) - 1
				} else if state.cursorGroup >= 0 && state.cursorGroup < len(state.options) {
					maxIdx = len(state.options[state.cursorGroup]) - 1
				}
				if state.cursorIdx > maxIdx {
					state.cursorIdx = maxIdx
				}

			case "space":
				if state.cursorGroup == -1 {
					state.common[state.cursorIdx].selected = !state.common[state.cursorIdx].selected
				} else if state.cursorGroup >= 0 {
					if state.activeOpt != -1 && state.activeOpt != state.cursorGroup {
						for i := range state.options[state.activeOpt] {
							state.options[state.activeOpt][i].selected = false
						}
					}
					state.activeOpt = state.cursorGroup
					state.options[state.cursorGroup][state.cursorIdx].selected = !state.options[state.cursorGroup][state.cursorIdx].selected

					hasActive := false
					for _, t := range state.options[state.cursorGroup] {
						if t.selected {
							hasActive = true
							break
						}
					}
					if !hasActive {
						state.activeOpt = -1
					}
				}

			case "enter":
				for _, t := range state.common {
					if t.selected && !slices.Contains(m.pendingTags[item.Name], t.name) {
						m.pendingTags[item.Name] = append(m.pendingTags[item.Name], t.name)
					}
				}
				for _, group := range state.options {
					for _, t := range group {
						if t.selected && !slices.Contains(m.pendingTags[item.Name], t.name) {
							m.pendingTags[item.Name] = append(m.pendingTags[item.Name], t.name)
						}
					}
				}

				sort.Strings(m.pendingTags[item.Name])
				m.implicitPendingTags[item.Name], _ = getImplicitTags(m.socketPath, m.pendingTags[item.Name])
				if m.cursor < len(m.items)-1 {
					m.cursor++
				}
			}
			return m, nil
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
				line += rowDimStyle.Render(fmt.Sprintf("  %s %d", iconTag, len(tags)))
			}

			if i == m.cursor {
				cursor = iconChevron + " "
				line = rowSelectedStyle.Render(fmt.Sprintf(" %s %s ", icon, item.Name))
				if tags := m.pendingTags[item.Name]; len(tags) > 0 {
					line += " " + rowDimStyle.Render(fmt.Sprintf("%s %d", iconTag, len(tags)))
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
		info += " | last modified: " + item.ModifiedAt
	}
	if item.Size != 0 {
		info += " | size: " + formatBytes(item.Size)
	}
	if !item.IsDir && item.MimeType != "" {
		info += " | mime type: " + item.MimeType
	}

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorText).Render(icon+"  "+item.Name) + "\n")
	b.WriteString(rowDimStyle.Render(info) + "\n\n")

	b.WriteString(sectionLabelStyle.Render(iconTags+" PENDING TAGS") + "\n")
	if tags := m.pendingTags[item.Name]; len(tags) > 0 {
		chips := make([]string, len(tags))
		for i, t := range tags {
			chips[i] = tagChipStyle.Render(iconTag + " " + t)
		}
		if implicitTags := m.implicitPendingTags[item.Name]; len(implicitTags) > 0 {
			for _, t := range implicitTags {
				chips = append(chips, implicitTagChipStyle.Render(iconMagic+" "+t))
			}
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, chips...) + "\n")
	} else {
		b.WriteString(emptyHintStyle.Render("no tags yet :p add one below") + "\n")
	}

	b.WriteString("\n" + sectionLabelStyle.Render(iconMagic+" SUGGESTED TAGS") + "\n")

	state, hasSuggestions := m.suggestions[item.Name]
	if !hasSuggestions {
		b.WriteString(emptyHintStyle.Render("no suggestions available") + "\n\n")
	} else {
		if len(state.common) > 0 {
			b.WriteString(rowDimStyle.Render("Common:") + "\n")
			var chips []string
			for i, tag := range state.common {
				isFocused := m.focus == focusSuggested && state.cursorGroup == -1 && state.cursorIdx == i
				chips = append(chips, renderChip(tag, isFocused, false))
			}
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, chips...) + "\n")
		}

		for gIdx, optGroup := range state.options {
			isDisabled := state.activeOpt != -1 && state.activeOpt != gIdx

			label := rowDimStyle
			if isDisabled {
				label = label.Faint(true)
			}
			b.WriteString(label.Render(fmt.Sprintf("Option %d:", gIdx+1)) + "\n")

			var chips []string
			for i, tag := range optGroup {
				isFocused := m.focus == focusSuggested && state.cursorGroup == gIdx && state.cursorIdx == i
				chips = append(chips, renderChip(tag, isFocused, isDisabled))
			}
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, chips...) + "\n")
		}
		b.WriteString("\n")
	}

	inputStyle := inputBoxStyle
	if m.focus == focusTagInput {
		inputStyle = inputBoxFocusedStyle
	}
	b.WriteString(sectionLabelStyle.Render(iconPlus+" ADD/REMOVE TAG") + "\n")
	b.WriteString(inputStyle.Width(width - 6).Render(m.tagInput.View()))

	style := panelStyle
	if m.focus == focusTagInput {
		style = focusedPanelStyle
	}
	return style.Width(width).Height(height).Render(b.String())
}

func (m model) footerView() string {
	div := dividerStyle.Render(strings.Repeat("─", max(m.width, 0)))

	applyBtn := buttonStyle.Foreground(colorGreen).Render(iconCheck + " Apply")
	cancelBtn := buttonStyle.Foreground(colorRed).MarginRight(0).Render(iconCross + " Cancel")

	switch m.focus {
	case focusApply:
		applyBtn = buttonStyle.Background(colorGreen).Foreground(colorBase).Bold(true).Render(iconCheck + " Apply")
	case focusCancel:
		cancelBtn = buttonStyle.MarginRight(0).Background(colorRed).Foreground(colorBase).Bold(true).Render(iconCross + " Cancel")
	default:
		break
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, applyBtn, cancelBtn)
	helpLine := m.help.View(m.keys)

	bottomRow := helpLine
	if m.toastMsg != "" {
		style := toastInfoStyle
		icon := iconInfo
		if m.toastIsErr {
			style = toastErrStyle
			icon = iconWarning
		}
		toastView := style.Render(icon + " " + m.toastMsg)

		gap := m.width - lipgloss.Width(helpLine) - lipgloss.Width(toastView)
		if gap > 0 {
			bottomRow += strings.Repeat(" ", gap)
		}
		bottomRow += toastView
	}

	return lipgloss.JoinVertical(lipgloss.Left, div, buttons, bottomRow)
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
