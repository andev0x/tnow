package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/andev0x/tnow/internal/config"
)

// fitLine truncates and pads an ANSI-aware line to exactly width cells.
func fitLine(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(line) > width {
		line = ansi.Truncate(line, width, "…")
	}
	if padding := width - ansi.StringWidth(line); padding > 0 {
		line += strings.Repeat(" ", padding)
	}
	return line
}

// wrapANSI wraps plain or styled text without breaking ANSI escape sequences.
func wrapANSI(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			lines = append(lines, "")
			continue
		}
		wrapped := ansi.Wrap(line, width, " ")
		lines = append(lines, strings.Split(wrapped, "\n")...)
	}
	return lines
}

// titledCard draws a rounded frame with the title centered on the top border.
// The content is clipped and padded to the requested height so the surrounding
// layout never jumps when text grows.
func (m Model) titledCard(title string, content []string, width, height int, focused bool) string {
	if width < 6 {
		width = 6
	}
	if height < 3 {
		height = 3
	}

	innerWidth := width - 2
	usableWidth := innerWidth - 2
	if usableWidth < 1 {
		usableWidth = 1
	}

	borderColor := ColorBorderSubtle
	if focused {
		borderColor = ColorBorderFocused
	}
	border := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor))

	title = strings.TrimSpace(title)
	maxTitleWidth := innerWidth - 2
	if maxTitleWidth < 1 {
		maxTitleWidth = 1
	}
	if ansi.StringWidth(title) > maxTitleWidth {
		title = ansi.Truncate(title, maxTitleWidth, "…")
	}
	titlePart := " " + m.styles.BoxLabel.Render(title) + " "
	titleWidth := ansi.StringWidth(titlePart)
	if titleWidth > innerWidth {
		titlePart = ansi.Truncate(titlePart, innerWidth, "")
		titleWidth = ansi.StringWidth(titlePart)
	}
	fill := innerWidth - titleWidth
	leftFill := fill / 2
	rightFill := fill - leftFill

	top := border.Render("╭") +
		border.Render(strings.Repeat("─", leftFill)) +
		titlePart +
		border.Render(strings.Repeat("─", rightFill)) +
		border.Render("╮")

	side := border.Render("│")
	var b strings.Builder
	b.WriteString(top)
	b.WriteByte('\n')
	for i := 0; i < height-2; i++ {
		line := ""
		if i < len(content) {
			line = content[i]
		}
		b.WriteString(side)
		b.WriteByte(' ')
		b.WriteString(fitLine(line, usableWidth))
		b.WriteByte(' ')
		b.WriteString(side)
		if i < height-3 {
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')
	b.WriteString(border.Render("╰") + border.Render(strings.Repeat("─", innerWidth)) + border.Render("╯"))
	return b.String()
}

// joinColumns combines two line slices with a fixed-width separator.
func joinColumns(left, right []string, leftWidth, rightWidth int, separator string) []string {
	height := max(len(left), len(right))
	lines := make([]string, height)
	for i := 0; i < height; i++ {
		leftLine, rightLine := "", ""
		if i < len(left) {
			leftLine = left[i]
		}
		if i < len(right) {
			rightLine = right[i]
		}
		lines[i] = fitLine(leftLine, leftWidth) + separator + fitLine(rightLine, rightWidth)
	}
	return lines
}

// prettyConfigPath abbreviates the user's home directory for display.
func prettyConfigPath() string {
	path := config.JSONPath()
	if path == "" {
		return "~/.tnow/config.json"
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		if path == home {
			return "~"
		}
		if strings.HasPrefix(path, home+string(os.PathSeparator)) {
			return "~" + strings.TrimPrefix(path, home)
		}
	}
	return path
}
