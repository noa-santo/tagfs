package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	suffixes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	value := float64(bytes)
	i := 0
	for value >= unit && i < len(suffixes)-1 {
		value /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", value, suffixes[i])
}

func remove[T comparable](l []T, item T) []T {
	out := make([]T, 0)
	for _, element := range l {
		if element != item {
			out = append(out, element)
		}
	}
	return out
}

func renderChip(tag selectableTag, isFocused, isDisabled bool) string {
	marker := "[ ]"
	if tag.selected {
		marker = "[x]"
	}
	label := fmt.Sprintf("%s %s", marker, tag.name)

	style := lipgloss.NewStyle().Padding(0, 1).MarginRight(1)

	switch {
	case isDisabled:
		style = style.Background(colorSurface).Foreground(colorOverlay)
	case isFocused && tag.selected:
		style = style.Background(colorGreen).Foreground(colorBase).Bold(true)
	case isFocused:
		style = style.Background(colorMauve).Foreground(colorBase).Bold(true)
	case tag.selected:
		style = style.Background(colorGreen).Foreground(colorBase)
	default:
		style = style.Background(colorSurface).Foreground(colorText)
	}

	return style.Render(label)
}
