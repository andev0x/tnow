package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"

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

// titledCard draws an opaque panel with a top-left title, an optional metadata
// label on the right side of the border, and a fixed grid. The title sits
// inset in the top border (╭─ Title ───╮) instead of being centered, which
// keeps the border continuous and reads like an IDE panel.
func (m Model) titledCard(title, meta string, content []string, width, height int, focused bool) string {
	if height < 3 {
		height = 3
	}
	// Below this width a border leaves no usable content cell. Degrade to a
	// clipped, background-painted text block instead of overflowing.
	if width < 6 {
		width = max(1, width)
		lines := make([]string, height)
		for i := 0; i < height; i++ {
			if i < len(content) {
				lines[i] = fitLine(content[i], width)
			} else {
				lines[i] = strings.Repeat(" ", width)
			}
		}
		return strings.Join(lines, "\n")
	}

	innerWidth := width - 2
	usableWidth := innerWidth - 2
	if usableWidth < 1 {
		usableWidth = 1
	}

	borderColor := ColorBorderIdle
	titleStyle := m.styles.PanelTitle
	if focused {
		borderColor = ColorBorderFocused
		titleStyle = m.styles.PanelTitleOn
	}
	border := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor))

	title = strings.TrimSpace(title)
	maxTitleWidth := innerWidth - 5
	if maxTitleWidth < 1 {
		maxTitleWidth = 1
	}
	if ansi.StringWidth(title) > maxTitleWidth {
		title = ansi.Truncate(title, maxTitleWidth, "…")
	}
	left := border.Render("─ ") + titleStyle.Render(title) + border.Render(" ")
	leftWidth := ansi.StringWidth(left)

	right := border.Render("─")
	rightWidth := 1
	meta = strings.TrimSpace(meta)
	if meta != "" {
		maxMetaWidth := innerWidth - leftWidth - 5
		if maxMetaWidth >= 4 {
			if ansi.StringWidth(meta) > maxMetaWidth {
				meta = ansi.Truncate(meta, maxMetaWidth, "…")
			}
			metaPart := border.Render(" ") + m.styles.PanelMeta.Render(meta) + border.Render(" ─")
			right = metaPart
			rightWidth = ansi.StringWidth(metaPart)
		}
	}

	fill := innerWidth - leftWidth - rightWidth
	if fill < 1 {
		// In a very narrow pane the metadata is the first thing to go.
		right = border.Render("─")
		rightWidth = 1
		fill = innerWidth - leftWidth - rightWidth
	}
	if fill < 0 {
		over := -fill
		title = ansi.Truncate(title, max(1, ansi.StringWidth(title)-over), "")
		left = border.Render("─ ") + titleStyle.Render(title) + border.Render(" ")
		leftWidth = ansi.StringWidth(left)
		fill = max(0, innerWidth-leftWidth-rightWidth)
	}

	top := border.Render("╭") +
		left +
		border.Render(strings.Repeat("─", fill)) +
		right +
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

// screenLayer is one opaque layer in the cell canvas. Text is drawn at the
// rectangle origin and clipped to its bounds. A layer with no text paints a
// solid rectangle, which is used for the floating palette's shadow.
type screenLayer struct {
	x, y          int
	width, height int
	text          string
	bg            string
}

// renderScreen composes all layers into a full-screen, opaque cell grid.
// Painting backgrounds at the cell level is what prevents terminal text or a
// lower panel from bleeding through a focused input, a popup, or a dialog.
func renderScreen(width, height int, baseBG string, layers ...screenLayer) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	buf := cellbuf.NewBuffer(width, height)
	fillBackground(buf, baseBG, cellbuf.Rect(0, 0, width, height))

	for _, layer := range layers {
		if layer.width <= 0 || layer.height <= 0 {
			continue
		}
		x := max(0, min(layer.x, width))
		y := max(0, min(layer.y, height))
		w := min(layer.width, width-x)
		h := min(layer.height, height-y)
		if w <= 0 || h <= 0 {
			continue
		}
		rect := cellbuf.Rect(x, y, w, h)
		if layer.text == "" {
			// A textless layer is a solid fill, used for shadows/backdrops.
			fillBackground(buf, layer.bg, rect)
			continue
		}
		cellbuf.SetContentRect(buf, layer.text, rect)
		stainBackground(buf, rect, layer.bg)
	}

	return renderCellBuffer(buf, width, baseBG)
}

// fillBackground paints a rectangle with an explicit background color.
func fillBackground(buf *cellbuf.Buffer, hex string, rect cellbuf.Rectangle) {
	rgb := hexRGB(hex)
	cell := &cellbuf.Cell{
		Rune:  ' ',
		Width: 1,
		Style: cellbuf.Style{Bg: rgb},
	}
	cellbuf.FillRect(buf, cell, rect)
}

// stainBackground gives every transparent cell an explicit background while
// preserving intentional backgrounds such as the selected command strip.
func stainBackground(buf *cellbuf.Buffer, rect cellbuf.Rectangle, hex string) {
	rgb := hexRGB(hex)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		if y < 0 || y >= len(buf.Lines) {
			continue
		}
		line := buf.Lines[y]
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if x < 0 || x >= len(line) {
				continue
			}
			cell := line[x]
			if cell == nil {
				cell = &cellbuf.Cell{Rune: ' ', Width: 1}
				buf.SetCell(x, y, cell)
			}
			if cell.Style.Bg == nil {
				cell.Style.Bg = rgb
			}
		}
	}
}

// renderCellBuffer turns the cell grid into terminal text. Rows are padded to
// the full width with an explicit background so no terminal default cells are
// left exposed at the edges.
func renderCellBuffer(buf *cellbuf.Buffer, width int, baseBG string) string {
	lines := make([]string, 0, buf.Height())
	for y := 0; y < buf.Height(); y++ {
		_, line := cellbuf.RenderLine(buf, y)
		line = ansi.Truncate(line, width, "")
		if pad := width - ansi.StringWidth(line); pad > 0 {
			line += backgroundSequence(baseBG) + strings.Repeat(" ", pad)
		}
		// Reset at every row boundary so a colored cell can never leak its
		// style into the next terminal line.
		lines = append(lines, line+ansi.ResetStyle)
	}
	return strings.Join(lines, "\n")
}

// hexRGB converts a palette hex value to a cellbuf-compatible true color.
func hexRGB(hex string) ansi.RGBColor {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) != 6 {
		return ansi.RGBColor{}
	}
	value, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return ansi.RGBColor{}
	}
	return ansi.RGBColor{
		R: uint8(value >> 16),
		G: uint8(value >> 8),
		B: uint8(value),
	}
}

// backgroundSequence emits a raw 24-bit background sequence for padding. The
// cell canvas already writes true-color cells, so keeping the padding in the
// same color space avoids edge artifacts in partial-width rows.
func backgroundSequence(hex string) string {
	rgb := hexRGB(hex)
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", rgb.R, rgb.G, rgb.B)
}

// centeredOverlay places a dialog over an opaque full-screen canvas.
func (m Model) centeredOverlay(box, bg string) string {
	width := m.width
	if width == 0 {
		width = 80
	}
	height := m.effectiveHeight()
	boxWidth, boxHeight := lipgloss.Size(box)
	if boxWidth <= 0 || boxHeight <= 0 {
		return renderScreen(width, height, ColorBgBase)
	}
	x := max(0, (width-boxWidth)/2)
	y := max(0, (height-boxHeight)/2)

	layers := []screenLayer{
		{x: x + 1, y: y + 1, width: boxWidth, height: boxHeight, bg: ColorBgShadow},
		{x: x, y: y, width: boxWidth, height: boxHeight, text: box, bg: bg},
	}
	return renderScreen(width, height, ColorBgBase, layers...)
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
