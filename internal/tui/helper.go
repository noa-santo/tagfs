package tui

import "fmt"

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

	style := tagChipStyle
	if isDisabled {
		style = style.Faint(true).Foreground(colorSubtext)
	} else if isFocused && tag.selected {
		style = style.Background(colorGreen).Foreground(colorBase).Bold(true)
	} else if isFocused {
		style = style.Background(colorMauve).Foreground(colorBase)
	} else if tag.selected {
		style = style.Foreground(colorGreen).Foreground(colorBase)
	}

	return style.Render(fmt.Sprintf("%s %s", marker, tag.name))
}
