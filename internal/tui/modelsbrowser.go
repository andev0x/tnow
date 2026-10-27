package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/andev0x/tnow/internal/app"
	"github.com/andev0x/tnow/internal/config"
)

// modelsModalHeight returns a fixed browser height for the current terminal.
// It deliberately does not depend on the number of providers or models, so
// filtering or switching catalogs never resizes the dialog.
func (m Model) modelsModalHeight() int {
	height := m.effectiveHeight()
	maxHeight := max(5, height-2)
	modalHeight := min(26, maxHeight)
	if height >= 12 && modalHeight < 12 {
		modalHeight = min(12, maxHeight)
	}
	return modalHeight
}

// renderModelsBrowser draws the provider/model control surface. Providers are
// always the left scope and models are always the right scope; the active
// model is pinned first in the right pane. The frame size is fixed, and long
// lists are windowed with an overflow marker instead of growing the box.
func (m Model) renderModelsBrowser() string {
	width := m.width
	if width == 0 {
		width = 80
	}
	height := m.effectiveHeight()

	modalWidth := max(24, min(96, width-4))
	if modalWidth > width-2 {
		modalWidth = max(10, width-2)
	}
	modalHeight := m.modelsModalHeight()
	innerWidth := max(1, modalWidth-4)
	contentLines := max(1, modalHeight-2)
	isNarrow := modalWidth < 58

	providers := m.providerNames()
	if len(providers) == 0 {
		body := []string{
			"",
			fitLine(m.styles.EmptyState.Render("No providers configured."), innerWidth),
			"",
			fitLine(m.styles.BoxMeta.Render("Press Esc to close."), innerWidth),
		}
		box := m.titledCard("MODEL CONTROL", body, modalWidth, modalHeight, true)
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
	}

	m.mbProviderIdx = max(0, min(m.mbProviderIdx, len(providers)-1))
	selectedProvider := providers[m.mbProviderIdx]
	models := m.mbFilteredModels(selectedProvider)
	if len(models) > 0 {
		m.mbModelIdx = max(0, min(m.mbModelIdx, len(models)-1))
	} else {
		m.mbModelIdx = 0
	}

	selectedModel := ""
	if len(models) > 0 {
		selectedModel = models[m.mbModelIdx]
	}

	m.mbSearchInput.Width = max(8, innerWidth-4)
	searchView := m.mbSearchInput.View()
	searchLine := fitLine(searchView, innerWidth)
	if m.filterExcellent {
		searchLine = alignEnds(searchView, m.styles.BadgeExcellent.Render("✦ optimized"), innerWidth)
	}

	// Fixed vertical structure: search, blank, panes, optional details,
	// optional notice, optional hint. The decisions depend only on the
	// terminal height, never on the selected provider/model.
	details := m.modelDetailLines(selectedModel, selectedProvider, innerWidth)
	showDetails := contentLines >= 11
	showNotice := contentLines >= 6
	showHint := contentLines >= 7

	reserved := 2 // search + blank
	if showDetails {
		reserved += len(details)
	}
	if showNotice {
		reserved++
	}
	if showHint {
		reserved++
	}
	columnHeight := max(2, contentLines-reserved)

	var body []string
	if isNarrow {
		providerHeight := min(max(2, columnHeight/3), 6)
		modelHeight := columnHeight - providerHeight - 1
		if modelHeight < 1 {
			providerHeight = max(1, columnHeight-2)
			modelHeight = 1
		}
		providerLines := m.providerPaneLines(providers, innerWidth, providerHeight)
		modelLines := m.modelPaneLines(models, selectedProvider, innerWidth, modelHeight)
		divider := m.styles.SubtleLine.Render(strings.Repeat("─", innerWidth))
		body = append(body, providerLines...)
		body = append(body, divider)
		body = append(body, modelLines...)
	} else {
		leftWidth := max(16, min(28, innerWidth/3))
		rightWidth := max(8, innerWidth-leftWidth-3)
		providerLines := m.providerPaneLines(providers, leftWidth, columnHeight)
		modelLines := m.modelPaneLines(models, selectedProvider, rightWidth, columnHeight)
		separator := " " + m.styles.SubtleLine.Render("│") + " "
		body = joinColumns(providerLines, modelLines, leftWidth, rightWidth, separator)
	}

	lines := make([]string, 0, contentLines)
	lines = append(lines, searchLine)
	lines = append(lines, "")
	lines = append(lines, body...)
	if showDetails {
		lines = append(lines, details...)
	}
	if showNotice {
		notice := m.modelsNotice(innerWidth)
		if notice == "" {
			notice = ""
		}
		lines = append(lines, notice)
	}
	if showHint {
		lines = append(lines, m.modelsHintLine(selectedProvider, selectedModel, innerWidth))
	}

	box := m.titledCard("MODEL CONTROL", lines, modalWidth, modalHeight, true)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// providerPaneLines returns exactly height lines. When the provider list is
// longer than the pane, the last line becomes a compact overflow marker.
func (m Model) providerPaneLines(providers []string, width, height int) []string {
	if height < 1 {
		height = 1
	}
	title := "PROVIDERS"
	if m.mbFocus == "provider" {
		title = m.styles.SuggestActive.Render("▸ " + title)
	} else {
		title = m.styles.BoxLabel.Render(title)
	}
	lines := []string{alignEnds(title, m.styles.BoxMeta.Render(fmt.Sprintf("%d", len(providers))), width)}

	listRows := height - len(lines)
	if listRows < 1 {
		return padLines(lines, height)
	}
	overflow := len(providers) > listRows
	if overflow {
		listRows--
	}
	if listRows < 1 {
		return padLines(lines, height)
	}

	start := 0
	if m.mbProviderIdx >= listRows {
		start = m.mbProviderIdx - listRows + 1
	}
	if start+listRows > len(providers) {
		start = max(0, len(providers)-listRows)
	}
	end := min(len(providers), start+listRows)

	for i := start; i < end; i++ {
		provider := providers[i]
		selected := i == m.mbProviderIdx
		var marker string
		switch {
		case selected && m.mbFocus == "provider":
			marker = "▸ "
		case selected:
			marker = "› "
		case provider == m.app.Provider:
			marker = "● "
		case !config.ProviderRequiresKey(provider) || m.app.HasAPIKey(provider):
			marker = "✓ "
		default:
			marker = "· "
		}
		modelCount := fmt.Sprintf("%d", len(m.modelsForProvider(provider)))
		maxName := width - lipgloss.Width(marker) - lipgloss.Width(modelCount) - 1
		if maxName < 4 {
			maxName = 4
		}
		name := ansi.Truncate(provider, maxName, "…")
		plain := marker + name + strings.Repeat(" ", max(1, maxName-ansi.StringWidth(name))) + " " + modelCount
		plain = fitLine(plain, width)
		switch {
		case selected && m.mbFocus == "provider":
			plain = m.styles.SuggestActive.Render(plain)
		case selected:
			plain = m.styles.SuggestIdle.Render(plain)
		case provider == m.app.Provider:
			plain = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess)).Render(plain)
		case !config.ProviderRequiresKey(provider) || m.app.HasAPIKey(provider):
			plain = fitLine(plain, width)
		default:
			plain = m.styles.BoxMeta.Render(plain)
		}
		lines = append(lines, plain)
	}
	if overflow {
		lines = append(lines, overflowLine(start, end, len(providers), width, &m.styles))
	}
	return padLines(lines, height)
}

// modelPaneLines returns exactly height lines. Long model catalogs scroll and
// end with an overflow marker rather than expanding the browser.
func (m Model) modelPaneLines(models []string, provider string, width, height int) []string {
	if height < 1 {
		height = 1
	}
	title := "MODELS · " + strings.ToLower(provider)
	if m.mbFocus == "model" {
		title = m.styles.SuggestActive.Render("▸ " + title)
	} else {
		title = m.styles.BoxLabel.Render(title)
	}
	position := ""
	if len(models) > 0 {
		position = fmt.Sprintf("%d/%d", m.mbModelIdx+1, len(models))
	}
	lines := []string{alignEnds(title, m.styles.BoxMeta.Render(position), width)}

	listRows := height - len(lines)
	if listRows < 1 {
		return padLines(lines, height)
	}
	if len(models) == 0 {
		message := "no models · Ctrl+E key · Esc close"
		if m.modelErr != "" {
			message = "catalog unavailable · Enter retry"
		}
		lines = append(lines, fitLine(m.styles.EmptyState.Render(message), width))
		return padLines(lines, height)
	}

	overflow := len(models) > listRows
	if overflow {
		listRows--
	}
	if listRows < 1 {
		return padLines(lines, height)
	}

	start := 0
	if m.mbModelIdx >= listRows {
		start = m.mbModelIdx - listRows + 1
	}
	if start+listRows > len(models) {
		start = max(0, len(models)-listRows)
	}
	end := min(len(models), start+listRows)

	for i := start; i < end; i++ {
		model := models[i]
		info := app.ModelInfoFor(model)
		context := app.FormatContextTokens(info.ContextTokens)
		capability := shortCapabilities(info)

		marker := "  "
		switch {
		case i == m.mbModelIdx && m.mbFocus == "model":
			marker = "▸ "
		case i == m.mbModelIdx:
			marker = "› "
		case model == m.app.Model && provider == m.app.Provider:
			marker = "● "
		}

		var plain string
		if width < 34 {
			maxName := width - lipgloss.Width(marker)
			if maxName < 4 {
				maxName = 4
			}
			plain = marker + fitLine(ansi.Truncate(model, maxName, "…"), maxName)
		} else {
			capWidth := 3
			contextWidth := 5
			maxName := width - lipgloss.Width(marker) - contextWidth - capWidth - 2
			if maxName < 8 {
				maxName = 8
			}
			name := ansi.Truncate(model, maxName, "…")
			plain = marker + fitLine(name, maxName) + " " + fitLine(context, contextWidth) + " " + fitLine(capability, capWidth)
		}
		plain = fitLine(plain, width)
		switch {
		case i == m.mbModelIdx && m.mbFocus == "model":
			plain = m.styles.SuggestActive.Render(plain)
		case i == m.mbModelIdx:
			plain = m.styles.SuggestIdle.Render(plain)
		case model == m.app.Model && provider == m.app.Provider:
			plain = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess)).Render(plain)
		default:
			plain = fitLine(plain, width)
		}
		lines = append(lines, plain)
	}
	if overflow {
		lines = append(lines, overflowLine(start, end, len(models), width, &m.styles))
	}
	return padLines(lines, height)
}

// modelDetailLines always returns three lines so the detail area keeps a
// constant height while the selection moves.
func (m Model) modelDetailLines(model, provider string, width int) []string {
	if strings.TrimSpace(model) == "" {
		return []string{
			"",
			fitLine(m.styles.BoxMeta.Render("No model selected"), width),
			fitLine(m.styles.BoxMeta.Render("Context —  ·  Vision —  ·  Tools —"), width),
		}
	}
	info := app.ModelInfoFor(model)
	context := app.FormatContextTokens(info.ContextTokens)
	vision := "—"
	tools := "—"
	if info.Known {
		vision = "no"
		tools = "no"
		if info.Vision {
			vision = "yes"
		}
		if info.Tools {
			tools = "yes"
		}
	}
	summary := fmt.Sprintf("Context %s  ·  Vision %s  ·  Tools %s", context, vision, tools)
	if width < 46 {
		summary = fmt.Sprintf("ctx %s · vision %s · tools %s", context, vision, tools)
	}
	title := m.styles.HighlightBold.Render(model)
	if model == m.app.Model && provider == m.app.Provider {
		title += " " + m.styles.BadgeExcellent.Render("(active)")
	}
	return []string{
		"",
		fitLine(title, width),
		fitLine(m.styles.BoxMeta.Render(summary), width),
	}
}

func (m Model) modelsNotice(width int) string {
	status := strings.ReplaceAll(m.status, "\n", " ")
	switch {
	case strings.HasPrefix(status, "key saved for "):
		return fitLine(m.styles.StatusSuccess.Render("✓ "+status), width)
	case strings.HasPrefix(status, "provider ") && strings.Contains(status, " · model "):
		return fitLine(m.styles.StatusSuccess.Render("✓ "+status), width)
	}
	return ""
}

func (m Model) modelsHintLine(provider, model string, width int) string {
	left := "←→ pane  ·  ↑↓ select  ·  Enter assign  ·  Ctrl+E key  ·  Ctrl+T filter  ·  Esc close"
	if width < 62 {
		left = "↑↓ select · Enter assign · Ctrl+E key · Esc close"
	}
	if width < 40 {
		left = "↑↓ · Enter · Ctrl+E · Esc"
	}
	right := "live catalog"
	if m.modelErr != "" {
		right = "catalog unavailable"
	}
	if strings.TrimSpace(model) == "" && strings.TrimSpace(provider) != "" {
		if config.ProviderRequiresKey(provider) && !m.app.HasAPIKey(provider) {
			right = "Ctrl+E add key"
		} else {
			right = "Enter to load models"
		}
	}
	if width < 52 {
		right = ""
	}
	line := alignEnds(m.styles.BoxMeta.Render(left), m.styles.BoxMeta.Render(right), width)
	return fitLine(line, width)
}

func shortCapabilities(info app.ModelInfo) string {
	if !info.Known {
		return "—"
	}
	switch {
	case info.Vision && info.Tools:
		return "V T"
	case info.Vision:
		return "V"
	case info.Tools:
		return "T"
	default:
		return "—"
	}
}

// overflowLine reports how many rows are hidden above/below the window.
func overflowLine(start, end, total, width int, styles *Styles) string {
	var text string
	switch {
	case start > 0 && end < total:
		text = fmt.Sprintf("… %d above, %d below", start, total-end)
	case start > 0:
		text = fmt.Sprintf("… %d above", start)
	default:
		text = fmt.Sprintf("… %d more", total-end)
	}
	return fitLine(styles.BoxMeta.Render("  "+text), width)
}

func padLines(lines []string, height int) []string {
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return lines
}

// alignEnds places right at the end of a width-sized line while keeping left.
func alignEnds(left, right string, width int) string {
	if right == "" {
		return fitLine(left, width)
	}
	leftWidth := ansi.StringWidth(left)
	rightWidth := ansi.StringWidth(right)
	if leftWidth+rightWidth+1 > width {
		return fitLine(left, width)
	}
	gap := width - leftWidth - rightWidth
	return fitLine(left+strings.Repeat(" ", gap)+right, width)
}
