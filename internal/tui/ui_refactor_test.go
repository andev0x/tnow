package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func forceTrueColor(t *testing.T) {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
}

func sizedModel(t *testing.T, width, height int) Model {
	t.Helper()
	m := testModelWithProviders(t)
	tm, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return tm.(Model)
}

func rawLines(view string) []string {
	return strings.Split(view, "\n")
}

func plainLines(view string) []string {
	lines := rawLines(view)
	for i, line := range lines {
		lines[i] = ansi.Strip(line)
	}
	return lines
}

func TestScreenPaintsEveryRowOpaque(t *testing.T) {
	m := sizedModel(t, 80, 24)
	view := m.View()
	base := backgroundSequence(ColorBgBase)
	panel := backgroundSequence(ColorBgPanel)

	for i, line := range rawLines(view) {
		if !strings.Contains(line, "\x1b[48;2;") {
			t.Errorf("line %d has no explicit background: %q", i, ansi.Strip(line))
		}
		if got := ansi.StringWidth(line); got != m.width {
			t.Errorf("line %d width = %d, want %d", i, got, m.width)
		}
		if !strings.Contains(line, base) && !strings.Contains(line, panel) {
			t.Errorf("line %d is missing an app background layer", i)
		}
	}

	plain := plainLines(view)
	inputLine, outputLine := -1, -1
	for i, line := range plain {
		if strings.Contains(line, "╭─ Input") {
			inputLine = i
		}
		if strings.Contains(line, "╭─ Translation") {
			outputLine = i
		}
	}
	if inputLine < 0 || outputLine < 0 {
		t.Fatalf("expected both panels:\n%s", strings.Join(plain, "\n"))
	}
	if !strings.Contains(rawLines(view)[inputLine], panel) {
		t.Error("input panel row does not use the opaque panel background")
	}
	if !strings.Contains(rawLines(view)[outputLine], panel) {
		t.Error("output panel row does not use the opaque panel background")
	}
}

func TestFocusMovesAccentToActivePanel(t *testing.T) {
	forceTrueColor(t)
	m := sizedModel(t, 80, 24)
	accent := "38;2;125;207;255" // ColorBorderFocused

	view := m.View()
	inputLine := lineContaining(view, "╭─ Input")
	outputLine := lineContaining(view, "╭─ Translation")
	if !strings.Contains(inputLine, accent) {
		t.Error("input border should carry the focus accent by default")
	}
	if strings.Contains(outputLine, accent) {
		t.Error("output border should be idle while the input is focused")
	}

	m.focusOutput = true
	view = m.View()
	inputLine = lineContaining(view, "╭─ Input")
	outputLine = lineContaining(view, "╭─ Translation")
	if strings.Contains(inputLine, accent) {
		t.Error("input border should be idle while the output is focused")
	}
	if !strings.Contains(outputLine, accent) {
		t.Error("output border should carry the focus accent when focused")
	}
}

func lineContaining(view, needle string) string {
	for _, line := range rawLines(view) {
		if strings.Contains(ansi.Strip(line), needle) {
			return line
		}
	}
	return ""
}

func TestFooterOnlyShowsContextShortcuts(t *testing.T) {
	m := sizedModel(t, 100, 30)

	idle := ansi.Strip(m.View())
	for _, want := range []string{"Alt+Enter Translate", "Commands", "Mode", "Quit"} {
		if !strings.Contains(idle, want) {
			t.Errorf("idle footer missing %q:\n%s", want, idle)
		}
	}
	if strings.Contains(idle, "Ctrl+E") || strings.Contains(idle, "Ctrl+H") {
		t.Errorf("idle footer exposes non-contextual shortcuts:\n%s", idle)
	}

	m.input.SetValue("/")
	m.input.CursorEnd()
	menu := ansi.Strip(m.View())
	for _, want := range []string{"Navigate", "Select", "Close"} {
		if !strings.Contains(menu, want) {
			t.Errorf("menu footer missing %q:\n%s", want, menu)
		}
	}
	if strings.Contains(menu, "Ctrl+D") {
		t.Errorf("menu footer should not show the quit shortcut:\n%s", menu)
	}

	m.input.SetValue("")
	m.loading = true
	m.status = "translating..."
	loading := ansi.Strip(m.View())
	for _, want := range []string{"Cancel", "Quit"} {
		if !strings.Contains(loading, want) {
			t.Errorf("loading footer missing %q:\n%s", want, loading)
		}
	}
	if strings.Contains(loading, "Copy") {
		t.Errorf("loading footer should not offer copy:\n%s", loading)
	}

	m.loading = false
	m.status = "done"
	m.result = "một bản dịch"
	m.focusOutput = false
	result := ansi.Strip(m.View())
	for _, want := range []string{"Copy", "Retry", "Commands"} {
		if !strings.Contains(result, want) {
			t.Errorf("result footer missing %q:\n%s", want, result)
		}
	}

	m.focusOutput = true
	focused := ansi.Strip(m.View())
	if !strings.Contains(focused, "Back") {
		t.Errorf("focused output footer should offer a way back:\n%s", focused)
	}
}

func TestCommandPaletteDoesNotMoveBaseLayout(t *testing.T) {
	m := sizedModel(t, 80, 24)
	base := m.View()
	inputRow := lineIndexContaining(base, "╭─ Input")
	outputRow := lineIndexContaining(base, "╭─ Translation")
	outputHeight := cardHeight(ansi.Strip(base), "Translation")

	m.input.SetValue("/")
	m.input.CursorEnd()
	menu := m.View()
	if got := lineIndexContaining(menu, "╭─ Input"); got != inputRow {
		t.Errorf("input row moved from %d to %d when the palette opened", inputRow, got)
	}
	if got := lineIndexContaining(menu, "╭─ Translation"); got != outputRow {
		t.Errorf("output row moved from %d to %d when the palette opened", outputRow, got)
	}
	if got := cardHeight(ansi.Strip(menu), "Translation"); got != outputHeight {
		t.Errorf("output card height changed from %d to %d", outputHeight, got)
	}
	if !strings.Contains(ansi.Strip(menu), "╭─ Commands") {
		t.Fatalf("command popup missing:\n%s", ansi.Strip(menu))
	}
	if rows := rawLines(menu); len(rows) != 24 {
		t.Errorf("menu view has %d rows, want 24", len(rows))
	}

	m.input.SetValue("/models")
	m.input.CursorEnd()
	filtered := m.View()
	if got := lineIndexContaining(filtered, "╭─ Input"); got != inputRow {
		t.Errorf("input row moved while filtering: %d", got)
	}
	if got := lineIndexContaining(filtered, "╭─ Translation"); got != outputRow {
		t.Errorf("output row moved while filtering: %d", got)
	}
	if cardHeight(ansi.Strip(menu), "Commands") != cardHeight(ansi.Strip(filtered), "Commands") {
		t.Error("popup height changed while filtering")
	}
}

func lineIndexContaining(view, needle string) int {
	for i, line := range rawLines(view) {
		if strings.Contains(ansi.Strip(line), needle) {
			return i
		}
	}
	return -1
}

func TestPopupUsesRaisedOpaqueBackground(t *testing.T) {
	m := sizedModel(t, 80, 24)
	m.input.SetValue("/")
	m.input.CursorEnd()
	view := m.View()
	popupLine := lineContaining(view, "╭─ Commands")
	if popupLine == "" {
		t.Fatal("command popup not found")
	}
	if !strings.Contains(popupLine, backgroundSequence(ColorBgRaised)) {
		t.Error("popup border does not use the raised opaque background")
	}
	if !strings.Contains(view, backgroundSequence(ColorBgShadow)) {
		t.Error("popup is missing its shadow layer")
	}
}

func TestLongOutputScrollsAndReportsRange(t *testing.T) {
	m := sizedModel(t, 60, 18)
	m.result = strings.Join([]string{
		"first line",
		strings.Repeat("unbroken-", 120),
		"third line",
		"fourth line",
		"fifth line",
		"sixth line",
		"seventh line",
		"eighth line",
		"ninth line",
		"tenth line",
	}, "\n")
	m.focusOutput = true
	view := m.View()
	plain := ansi.Strip(view)

	if !strings.Contains(plain, "/") {
		t.Errorf("output title should report a scroll range:\n%s", plain)
	}
	if !strings.Contains(plain, "Scroll") {
		t.Errorf("footer should offer scrolling for overflowing output:\n%s", plain)
	}
	if !m.resultOverflow() {
		t.Error("long result should be reported as overflowing")
	}
	for i, line := range rawLines(view) {
		if got := ansi.StringWidth(line); got != 60 {
			t.Errorf("line %d width = %d, want 60", i, got)
		}
	}
}

func TestLongUnbrokenInputKeepsPanelGrid(t *testing.T) {
	m := sizedModel(t, 80, 24)
	height := cardHeight(ansi.Strip(m.View()), "Input")
	m.input.SetValue(strings.Repeat("supercalifragilisticexpialidocious", 20))
	m.input.CursorEnd()
	view := m.View()
	if got := cardHeight(ansi.Strip(view), "Input"); got != height {
		t.Errorf("input card height = %d, want %d", got, height)
	}
	if !strings.Contains(ansi.Strip(view), "╭─ Input") {
		t.Error("input title disappeared with long text")
	}
	for i, line := range rawLines(view) {
		if got := ansi.StringWidth(line); got != 80 {
			t.Errorf("line %d width = %d, want 80", i, got)
		}
	}
}

func TestHeaderUsesSolidActiveTab(t *testing.T) {
	m := sizedModel(t, 80, 24)
	if m.styles.ModeActive.GetBackground() == nil {
		t.Error("active mode tab should use a solid background pill")
	}
	if _, none := m.styles.ModeInactive.GetBackground().(lipgloss.NoColor); !none {
		t.Error("inactive tabs should be transparent and should not compete with the active pill")
	}
	header := ansi.Strip(rawLines(m.View())[0])
	if !strings.Contains(header, "Quick") || !strings.Contains(header, "Analyze") {
		t.Error("both navigation tabs should be visible at this width")
	}
}

func TestLanguagePickerKeepsTargetPinned(t *testing.T) {
	m := sizedModel(t, 80, 24)
	m.input.SetValue("/to ")
	m.input.CursorEnd()
	_, items, _, _ := m.suggestionItems()
	if len(items) == 0 || items[0] != "vi" {
		t.Fatalf("target language should be pinned first, got %v", items)
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "vi · Vietnamese · Vietnam") {
		t.Errorf("pinned language is not visible in the popup:\n%s", view)
	}
}

func TestEscapeDismissesPaletteAndPreservesDraft(t *testing.T) {
	m := sizedModel(t, 80, 24)
	m.input.SetValue("hello\n/to viet")
	m.input.CursorEnd()
	if !m.hasSuggestions() {
		t.Fatal("expected the language palette to be open")
	}
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(Model)
	if m.hasSuggestions() {
		t.Fatal("Esc should close the palette")
	}
	if got := m.input.Value(); got != "hello" {
		t.Errorf("draft = %q, want hello", got)
	}
}

func TestPaletteUsesOutputSlotInShortPane(t *testing.T) {
	m := sizedModel(t, 50, 12)
	m.input.SetValue("/")
	m.input.CursorEnd()
	view := m.View()
	plain := ansi.Strip(view)
	if !strings.Contains(plain, "╭─ Input") {
		t.Fatalf("input panel disappeared in a short pane:\n%s", plain)
	}
	if !strings.Contains(plain, "╭─ Commands") {
		t.Fatalf("palette did not use the output slot:\n%s", plain)
	}
	if strings.Contains(plain, "╭─╭─") {
		t.Fatalf("popup borders collided with the base panel:\n%s", plain)
	}
	if rows := rawLines(view); len(rows) != 12 {
		t.Errorf("short-pane palette view has %d rows, want 12", len(rows))
	}
	for i, line := range rawLines(view) {
		if got := ansi.StringWidth(line); got != 50 {
			t.Errorf("line %d width = %d, want 50", i, got)
		}
	}
}

func TestImportantStatusSurvivesNarrowFooter(t *testing.T) {
	m := sizedModel(t, 40, 14)
	m.status = "provider failed: connection refused by upstream endpoint"
	view := m.View()
	plain := ansi.Strip(view)
	if !strings.Contains(plain, "✗") || !strings.Contains(plain, "connection") {
		t.Fatalf("important error was dropped from the narrow footer:\n%s", plain)
	}
	for i, line := range rawLines(view) {
		if got := ansi.StringWidth(line); got != 40 {
			t.Errorf("line %d width = %d, want 40", i, got)
		}
	}
}

func TestLayoutStableAcrossRuntimeStates(t *testing.T) {
	base := sizedModel(t, 80, 24)
	inputRow := lineIndexContaining(base.View(), "╭─ Input")
	outputRow := lineIndexContaining(base.View(), "╭─ Translation")
	inputHeight := cardHeight(ansi.Strip(base.View()), "Input")
	outputHeight := cardHeight(ansi.Strip(base.View()), "Translation")

	states := []struct {
		name  string
		apply func(Model) Model
	}{
		{"idle", func(m Model) Model { return m }},
		{"loading", func(m Model) Model {
			m.loading = true
			m.status = "translating..."
			m.focusOutput = true
			return m
		}},
		{"result", func(m Model) Model {
			m.result = "một bản dịch"
			m.status = "done"
			m.focusOutput = true
			return m
		}},
		{"error", func(m Model) Model {
			m.status = "provider failed: connection refused"
			return m
		}},
		{"commands", func(m Model) Model {
			m.input.SetValue("/")
			m.input.CursorEnd()
			return m
		}},
	}

	for _, state := range states {
		m := state.apply(base)
		view := m.View()
		if got := lineIndexContaining(view, "╭─ Input"); got != inputRow {
			t.Errorf("%s: input row = %d, want %d", state.name, got, inputRow)
		}
		if got := lineIndexContaining(view, "╭─ Translation"); got != outputRow && state.name != "commands" {
			t.Errorf("%s: output row = %d, want %d", state.name, got, outputRow)
		}
		if got := cardHeight(ansi.Strip(view), "Input"); got != inputHeight {
			t.Errorf("%s: input height = %d, want %d", state.name, got, inputHeight)
		}
		if got := cardHeight(ansi.Strip(view), "Translation"); got != outputHeight && state.name != "commands" {
			t.Errorf("%s: output height = %d, want %d", state.name, got, outputHeight)
		}
		if rows := rawLines(view); len(rows) != 24 {
			t.Errorf("%s: view rows = %d, want 24", state.name, len(rows))
		}
		if strings.TrimSpace(ansi.Strip(rawLines(view)[23])) == "" {
			t.Errorf("%s: footer is not on the last row", state.name)
		}
	}
}
