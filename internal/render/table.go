package render

import (
	"strings"
)

func Table(headers []string, rows [][]string) string {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, row := range rows {
		for col := 0; col < len(headers) && col < len(row); col++ {
			if len(row[col]) > widths[col] {
				widths[col] = len(row[col])
			}
		}
	}

	var b strings.Builder
	b.WriteString(formatRow(headers, widths))
	b.WriteByte('\n')
	b.WriteString(separator(widths))
	for _, row := range rows {
		b.WriteByte('\n')
		b.WriteString(formatRow(row, widths))
	}
	return b.String()
}

func Properties(props [][2]string) string {
	width := len("key")
	for _, prop := range props {
		if len(prop[0]) > width {
			width = len(prop[0])
		}
	}

	var b strings.Builder
	b.WriteString(padRight("key", width))
	b.WriteString(" | value")
	b.WriteByte('\n')
	b.WriteString(strings.Repeat("-", width))
	b.WriteString("-|------")
	for _, prop := range props {
		b.WriteByte('\n')
		b.WriteString(padRight(prop[0], width))
		b.WriteString(" | ")
		b.WriteString(prop[1])
	}
	return b.String()
}

func formatRow(values []string, widths []int) string {
	formatted := make([]string, len(widths))
	for i := range widths {
		value := ""
		if i < len(values) {
			value = values[i]
		}
		formatted[i] = padRight(value, widths[i])
	}
	return strings.Join(formatted, " | ")
}

func separator(widths []int) string {
	parts := make([]string, len(widths))
	for i, width := range widths {
		parts[i] = strings.Repeat("-", width)
	}
	return strings.Join(parts, "-|-")
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}
