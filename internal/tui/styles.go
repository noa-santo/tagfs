package tui

import "charm.land/lipgloss/v2"

const (
	iconInbox   = "\uf01c"
	iconFolder  = "\uf07b"
	iconFile    = "\uf15b"
	iconTag     = "\uf02b"
	iconTags    = "\uf02c"
	iconMagic   = "\uf0d0"
	iconCheck   = "\uf00c"
	iconCross   = "\uf00d"
	iconPlus    = "\uf067"
	iconChevron = "\uf054"
	iconWarning = "\uf071"
	iconInfo    = "\uf129"
)

var (
	colorBase    = lipgloss.Color("#1e1e2e")
	colorSurface = lipgloss.Color("#313244")
	colorOverlay = lipgloss.Color("#6c7086")
	colorText    = lipgloss.Color("#cdd6f4")
	colorSubtext = lipgloss.Color("#a6adc8")
	colorMauve   = lipgloss.Color("#cba6f7")
	colorBlue    = lipgloss.Color("#89b4fa")
	colorGreen   = lipgloss.Color("#a6e3a1")
	colorRed     = lipgloss.Color("#f38ba8")
	colorPeach   = lipgloss.Color("#fab387")

	titleStyle = lipgloss.NewStyle().
			Background(colorMauve).
			Foreground(colorBase).
			Bold(true).
			Padding(0, 2)

	subtitleStyle = lipgloss.NewStyle().Foreground(colorSubtext)
	countStyle    = lipgloss.NewStyle().Foreground(colorOverlay)

	dividerStyle = lipgloss.NewStyle().Foreground(colorSurface)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSurface).
			Padding(0, 1)

	focusedPanelStyle = panelStyle.
				BorderForeground(colorMauve)

	panelTitleStyle = lipgloss.NewStyle().Foreground(colorMauve).Bold(true)

	rowSelectedStyle = lipgloss.NewStyle().Foreground(colorBase).Background(colorMauve).Bold(true)
	rowDirStyle      = lipgloss.NewStyle().Foreground(colorBlue)
	rowFileStyle     = lipgloss.NewStyle().Foreground(colorText)
	rowDimStyle      = lipgloss.NewStyle().Foreground(colorOverlay)

	sectionLabelStyle = lipgloss.NewStyle().Foreground(colorSubtext).Bold(true)
	emptyHintStyle    = lipgloss.NewStyle().Foreground(colorOverlay).Italic(true)

	tagChipStyle = lipgloss.NewStyle().
			Background(colorMauve).
			Foreground(colorBase).
			Bold(true).
			Padding(0, 1).
			MarginRight(1)
	implicitTagChipStyle = tagChipStyle.Background(colorOverlay)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSurface).
			Padding(0, 1)

	inputBoxFocusedStyle = inputBoxStyle.BorderForeground(colorPeach)

	buttonStyle = lipgloss.NewStyle().
			Foreground(colorSubtext).
			Padding(0, 2).
			MarginRight(2)

	errStyle = lipgloss.NewStyle().Foreground(colorRed).Bold(true)

	toastBaseStyle = lipgloss.NewStyle().Padding(0, 1)
	toastErrStyle  = toastBaseStyle.Background(colorRed).Foreground(colorBase).Bold(true)
	toastInfoStyle = toastBaseStyle.Background(colorBlue).Foreground(colorBase).Bold(true)
)
