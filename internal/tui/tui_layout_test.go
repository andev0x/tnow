package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/andev0x/tnow/internal/app"
	"github.com/andev0x/tnow/internal/config"
	"github.com/andev0x/tnow/internal/language"
)

func testModelWithProviders(t *testing.T) Model {
	t.Helper()
	cfg := config.Default()
	cfg.DefaultProvider = "groq"
	cfg.DefaultModel = "llama-3.3-70b-versatile"
	cfg.Providers = []config.ProviderConfig{
		{Name: "groq", BaseURL: "https://api.groq.com/openai/v1", Model: "llama-3.3-70b-versatile"},
		{Name: "ollama", BaseURL: "http://localhost:11434", Model: "llama3.2"},
		{Name: "openai", BaseURL: "https://api.openai.com/v1"},
	}
	a, err := app.New(cfg, "", "")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	m := NewModel(a)
	tm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return tm.(Model)
}

func TestSlashMenuShowsAndSurvivesRootEnter(t *testing.T) {
	m := testModelWithProviders(t)
	m.input.SetValue("")
	m.input.CursorEnd()

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = tm.(Model)
	if !m.hasSuggestions() {
		t.Fatal("expected slash suggestions after typing /")
	}
	view := ansi.Strip(m.View())
	for _, want := range []string{"Commands", "/models", "/to <language|country>", "/quit"} {
		if !strings.Contains(view, want) {
			t.Errorf("expected slash menu to contain %q, got:\n%s", want, view)
		}
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(Model)
	if m.input.Value() != "/" {
		t.Errorf("Enter on bare / should not rewrite the input, got %q", m.input.Value())
	}
	if m.showModelsBrowser || m.loading {
		t.Errorf("Enter on bare / should not submit or open the browser")
	}
}

func TestSlashMenuWorksAfterMultilineText(t *testing.T) {
	m := testModelWithProviders(t)
	m.input.SetValue("hello\n")
	m.input.CursorEnd()

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = tm.(Model)
	if got := m.input.Value(); got != "hello\n/" {
		t.Fatalf("expected slash on the new line, got %q", got)
	}
	if !m.hasSuggestions() {
		t.Fatal("expected suggestions for a slash command on the last line")
	}
	if !strings.Contains(ansi.Strip(m.View()), "Commands") {
		t.Fatal("expected the command menu to be rendered")
	}
}

func TestSlashMenuVisibleInShortSplitPane(t *testing.T) {
	m := testModelWithProviders(t)
	tm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 18})
	m = tm.(Model)
	m.input.SetValue("")
	m.input.CursorEnd()
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = tm.(Model)

	view := ansi.Strip(m.View())
	for _, want := range []string{"Commands", "/models"} {
		if !strings.Contains(view, want) {
			t.Errorf("short split pane should still show %q:\n%s", want, view)
		}
	}
	if lines := strings.Split(view, "\n"); len(lines) > 18 {
		t.Errorf("short split pane view has %d lines, want <= 18", len(lines))
	}
}

func TestEnterOnModelsCommandOpensBrowser(t *testing.T) {
	for _, command := range []string{"/models", "/model", "/provider"} {
		m := testModelWithProviders(t)
		m.input.SetValue(command)
		m.input.CursorEnd()

		tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = tm.(Model)
		if !m.showModelsBrowser {
			t.Fatalf("expected %s to open the models browser", command)
		}
	}
}

func TestProviderCommandSelectsProvider(t *testing.T) {
	m := testModelWithProviders(t)
	m.input.SetValue("/provider openai")
	m.input.CursorEnd()

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(Model)
	if !m.showModelsBrowser {
		t.Fatal("expected /provider openai to open the models browser")
	}
	providers := m.providerNames()
	if m.mbProviderIdx < 0 || m.mbProviderIdx >= len(providers) || providers[m.mbProviderIdx] != "openai" {
		t.Fatalf("selected provider = %v, want openai", providers[m.mbProviderIdx])
	}
}

func TestSplitPaneLayoutFitsTerminal(t *testing.T) {
	sizes := []struct {
		width, height int
	}{
		{60, 18},
		{80, 20},
		{40, 14},
		{120, 40},
		{50, 12},
	}
	for _, size := range sizes {
		m := testModelWithProviders(t)
		tm, _ := m.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
		m = tm.(Model)
		view := m.View()
		lines := strings.Split(view, "\n")
		if len(lines) > size.height {
			t.Errorf("%dx%d view has %d lines, want <= %d", size.width, size.height, len(lines), size.height)
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size.width {
				t.Errorf("%dx%d view line width %d exceeds terminal width: %q", size.width, size.height, ansi.StringWidth(line), ansi.Strip(line))
			}
		}
		plain := ansi.Strip(view)
		if !strings.Contains(plain, "Input") {
			t.Errorf("%dx%d view is missing the INPUT title", size.width, size.height)
		}
		if size.height >= 18 && !strings.Contains(plain, "Translation") {
			t.Errorf("%dx%d view is missing the TRANSLATION title", size.width, size.height)
		}
	}
}

func assertTopLeftBorderTitle(t *testing.T, view, title string) {
	t.Helper()
	prefix := "╭─ " + title
	for _, line := range strings.Split(view, "\n") {
		plain := strings.TrimSpace(ansi.Strip(line))
		if strings.HasPrefix(plain, prefix) {
			return
		}
	}
	t.Errorf("could not find %q as a top-left border title", title)
}

func TestInputAndTranslationTitlesAreTopLeft(t *testing.T) {
	m := testModelWithProviders(t)
	view := ansi.Strip(m.View())
	assertTopLeftBorderTitle(t, view, "Input")
	assertTopLeftBorderTitle(t, view, "Translation")
}

func TestModelsBrowserSeparatesProvidersAndModels(t *testing.T) {
	m := testModelWithProviders(t)
	m.modelCache["groq"] = []string{"aaa-model", "zzz-model", "llama-3.3-70b-versatile"}
	m.openModelsBrowser()

	models := m.mbFilteredModels("groq")
	if len(models) == 0 || models[0] != "llama-3.3-70b-versatile" {
		t.Fatalf("active model should be first, got %v", models)
	}
	if got := m.modelIndexForProvider("groq"); got != 0 {
		t.Errorf("active model index = %d, want 0", got)
	}

	view := ansi.Strip(m.View())
	for _, want := range []string{"Providers", "Models · groq", "Context", "Vision", "Tools"} {
		if !strings.Contains(view, want) {
			t.Errorf("models browser missing %q:\n%s", want, view)
		}
	}
	for _, unwanted := range []string{"Price", "Role     ·", "Output   ·", "Thinking ·", "openrouter/"} {
		if strings.Contains(view, unwanted) {
			t.Errorf("models browser should not contain %q:\n%s", unwanted, view)
		}
	}
}

func TestModelsBrowserFitsSplitPane(t *testing.T) {
	sizes := []struct {
		width, height int
	}{
		{52, 24},
		{100, 30},
		{40, 18},
	}
	for _, size := range sizes {
		m := testModelWithProviders(t)
		tm, _ := m.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
		m = tm.(Model)
		m.openModelsBrowser()
		view := m.View()
		lines := strings.Split(view, "\n")
		if len(lines) > size.height {
			t.Errorf("%dx%d browser has %d lines, want <= %d", size.width, size.height, len(lines), size.height)
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size.width {
				t.Errorf("%dx%d browser line width %d exceeds terminal width: %q", size.width, size.height, ansi.StringWidth(line), ansi.Strip(line))
			}
		}
		if !strings.Contains(ansi.Strip(view), "Model Control") {
			t.Errorf("%dx%d browser is missing its title", size.width, size.height)
		}
	}
}

// cardHeight finds a titled box and returns its total height in rows.
func cardHeight(view, title string) int {
	lines := strings.Split(view, "\n")
	start := -1
	column := -1
	for i, line := range lines {
		plain := ansi.Strip(line)
		if strings.Contains(plain, title) && strings.Contains(plain, "╭") {
			start = i
			column = strings.Index(plain, "╭")
			break
		}
	}
	if start < 0 {
		return 0
	}
	for i := start; i < len(lines); i++ {
		plain := ansi.Strip(lines[i])
		if strings.Index(plain, "╰") == column {
			return i - start + 1
		}
	}
	return 0
}

func TestInputCardHeightIsStable(t *testing.T) {
	m := testModelWithProviders(t)
	// A reference result-card height before the slash menu replaces it.
	baseView := ansi.Strip(m.View())
	resultHeight := cardHeight(baseView, "Translation")
	if resultHeight == 0 {
		t.Fatal("expected a translation card at the default test size")
	}

	cases := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"short", "hello"},
		{"wrapped", strings.Repeat("a long sentence that wraps inside the fixed input box. ", 4)},
		{"multiline", "first\nsecond\nthird\nfourth\nfifth\nsixth\nseventh"},
		{"slash", "/"},
		{"slash-filtered", "/models"},
		{"language-menu", "/to "},
		{"language-filtered", "/to vietnam"},
	}

	var inputHeight int
	for _, tc := range cases {
		m.input.SetValue(tc.value)
		m.input.CursorEnd()
		view := ansi.Strip(m.View())

		got := cardHeight(view, "Input")
		if got == 0 {
			t.Fatalf("%s: input card not found:\n%s", tc.name, view)
		}
		if inputHeight == 0 {
			inputHeight = got
		} else if got != inputHeight {
			t.Errorf("%s: input card height = %d, want stable %d:\n%s", tc.name, got, inputHeight, view)
		}

		if strings.HasPrefix(strings.TrimSpace(tc.value), "/") {
			menuTitle := "Commands"
			if strings.HasPrefix(strings.TrimSpace(tc.value), "/to") {
				menuTitle = "Languages"
			}
			if got := cardHeight(view, menuTitle); got == 0 {
				t.Errorf("%s: suggestion popup missing:\n%s", tc.name, view)
			}
			if got := cardHeight(view, "Translation"); got != resultHeight {
				t.Errorf("%s: output card height = %d, want stable %d:\n%s", tc.name, got, resultHeight, view)
			}
			if !strings.Contains(view, "Navigate") || !strings.Contains(view, "Close") {
				t.Errorf("%s: footer must show menu-specific shortcuts:\n%s", tc.name, view)
			}
		} else if got := cardHeight(view, "Translation"); got != resultHeight {
			t.Errorf("%s: result card height = %d, want stable %d:\n%s", tc.name, got, resultHeight, view)
		}
	}
}

func TestEmptyInputShowsPlaceholder(t *testing.T) {
	m := testModelWithProviders(t)
	m.input.SetValue("")
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Type or paste text") {
		t.Fatalf("empty fixed input should show its placeholder:\n%s", view)
	}
	// Placeholder wrapping must stay inside the frame rather than widening it.
	for _, line := range strings.Split(view, "\n") {
		if ansi.StringWidth(line) > m.width {
			t.Fatalf("line exceeds terminal width: %q", ansi.Strip(line))
		}
	}
}

func TestModelsBrowserHeightIsStable(t *testing.T) {
	m := testModelWithProviders(t)
	m.openModelsBrowser()

	measure := func(models []string) (int, string) {
		m.modelCache["groq"] = models
		m.mbModelIdx = 0
		view := ansi.Strip(m.View())
		return cardHeight(view, "Model Control"), view
	}

	shortHeight, shortView := measure([]string{"llama-3.3-70b-versatile", "gemma2-9b-it"})
	if strings.Contains(shortView, "more") {
		t.Errorf("short model list should not show an overflow marker:\n%s", shortView)
	}

	many := make([]string, 0, 40)
	for i := 0; i < 40; i++ {
		many = append(many, fmt.Sprintf("model-%02d", i))
	}
	longHeight, longView := measure(many)
	if longHeight != shortHeight {
		t.Errorf("models browser height changed with catalog size: short=%d long=%d", shortHeight, longHeight)
	}
	if !strings.Contains(longView, "more") {
		t.Errorf("long model list should be truncated with an overflow marker:\n%s", longView)
	}
	if lines := strings.Split(longView, "\n"); len(lines) > m.height {
		t.Errorf("models browser has %d lines, want <= %d", len(lines), m.height)
	}

	m.modelCache["groq"] = many
	m.mbSearchInput.SetValue("model-39")
	filteredHeight, filteredView := measure(many)
	if filteredHeight != longHeight {
		t.Errorf("models browser height changed after filtering: %d vs %d", filteredHeight, longHeight)
	}
	if !strings.Contains(filteredView, "model-39") {
		t.Errorf("filtered model missing:\n%s", filteredView)
	}
}

func TestShortPaneSlashKeepsTypedLineVisible(t *testing.T) {
	m := testModelWithProviders(t)
	tm, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 14})
	m = tm.(Model)
	_ = m.View()

	typing := "one\ntwo\nthree\nfour\nfive\nsix\nseven\n/"
	for _, r := range typing {
		tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = tm.(Model)
		_ = m.View()
	}

	view := ansi.Strip(m.View())
	if !strings.Contains(view, "seven") || !strings.Contains(view, "/") {
		t.Fatalf("the slash line and its context should stay visible in a short pane:\n%s", view)
	}
	if got := cardHeight(view, "Input"); got == 0 {
		t.Fatalf("input card disappeared in a short pane:\n%s", view)
	}
}

func TestEnterOnToOpensLanguagePicker(t *testing.T) {
	m := testModelWithProviders(t)
	m.input.SetValue("/to")
	m.input.CursorEnd()
	_ = m.View()

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(Model)
	if m.input.Value() != "/to " {
		t.Fatalf("input = %q, want /to ", m.input.Value())
	}
	kind, items, _, _ := m.suggestionItems()
	if kind != "language" || len(items) == 0 {
		t.Fatalf("expected the language picker, got kind=%q items=%d", kind, len(items))
	}
}

func TestToLanguageSuggestions(t *testing.T) {
	m := testModelWithProviders(t)
	m.input.SetValue("/to ")
	m.input.CursorEnd()
	_ = m.View()

	kind, items, title, _ := m.suggestionItems()
	if kind != "language" {
		t.Fatalf("suggestion kind = %q, want language", kind)
	}
	if title != "Languages" {
		t.Errorf("suggestion title = %q, want Languages", title)
	}
	if len(items) != len(language.All()) {
		t.Errorf("language suggestions = %d, want %d", len(items), len(language.All()))
	}
	if len(items) == 0 || items[0] != string(language.VI) {
		t.Errorf("active target should be pinned first, got %v", items)
	}

	view := ansi.Strip(m.View())
	for _, want := range []string{"Languages", "vi · Vietnamese · Vietnam"} {
		if !strings.Contains(view, want) {
			t.Errorf("language menu missing %q:\n%s", want, view)
		}
	}
	if h := cardHeight(view, "Input"); h == 0 {
		t.Fatalf("input card missing:\n%s", view)
	}

	m.input.SetValue("/to ja")
	m.input.CursorEnd()
	kind, items, _, _ = m.suggestionItems()
	if kind != "language" || len(items) == 0 || items[0] != "ja" {
		t.Fatalf("language search = kind %q, items %v; want Japanese first", kind, items)
	}

	m.input.SetValue("/to mexi")
	m.input.CursorEnd()
	kind, items, _, _ = m.suggestionItems()
	if kind != "language" || len(items) == 0 || items[0] != "es" {
		t.Fatalf("country-name filtering = kind %q, items %v; want Spanish first", kind, items)
	}
}

func TestToCountryNameResolves(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TNOW_CONFIG", filepath.Join(dir, "config.yaml"))
	t.Setenv("TNOW_JSON_CONFIG", filepath.Join(dir, "config.json"))

	tests := []struct {
		input string
		want  language.Language
	}{
		{"/to vietnam", language.VI},
		{"/to việt nam", language.VI},
		{"/to japan", language.JA},
		{"/to nhật bản", language.JA},
		{"/to south korea", language.KO},
		{"/to taiwan", language.ZHTW},
		{"/to mexico", language.ES},
		{"/to brazil", language.PT},
		{"/to saudi arabia", language.AR},
		{"/to hong kong", language.YUE},
	}

	for _, tt := range tests {
		m := testModelWithProviders(t)
		m.input.SetValue(tt.input)
		m.input.CursorEnd()
		_ = m.View()

		tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = tm.(Model)
		if m.app.TargetLang != tt.want {
			t.Errorf("%q set target %q, want %q (status %q)", tt.input, m.app.TargetLang, tt.want, m.status)
		}
		if strings.Contains(m.status, "choose a language") {
			t.Errorf("%q should resolve without requiring a selection", tt.input)
		}
	}
}

func TestToTabCompletesLanguageCode(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TNOW_CONFIG", filepath.Join(dir, "config.yaml"))
	t.Setenv("TNOW_JSON_CONFIG", filepath.Join(dir, "config.json"))

	m := testModelWithProviders(t)
	m.input.SetValue("/to vie")
	m.input.CursorEnd()
	_ = m.View()

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = tm.(Model)
	if m.input.Value() != "/to vi" {
		t.Fatalf("Tab completion = %q, want /to vi", m.input.Value())
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(Model)
	if m.app.TargetLang != language.VI {
		t.Errorf("target after Tab+Enter = %q, want Vietnamese", m.app.TargetLang)
	}
}

func TestToAmbiguousQueryKeepsMenuOpen(t *testing.T) {
	m := testModelWithProviders(t)
	previous := m.app.TargetLang
	m.input.SetValue("/to united")
	m.input.CursorEnd()
	_ = m.View()

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(Model)
	if !m.hasSuggestions() {
		t.Fatal("ambiguous query should keep the language menu open")
	}
	if !strings.Contains(m.status, "choose a language") {
		t.Errorf("status = %q, want a choose-a-language hint", m.status)
	}
	if m.app.TargetLang != previous {
		t.Errorf("ambiguous query changed target from %q to %q", previous, m.app.TargetLang)
	}
}

func TestToCommandPreservesMultilineDraft(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TNOW_CONFIG", filepath.Join(dir, "config.yaml"))
	t.Setenv("TNOW_JSON_CONFIG", filepath.Join(dir, "config.json"))

	m := testModelWithProviders(t)
	m.input.SetValue("hello\n/to vietnam")
	m.input.CursorEnd()
	_ = m.View()

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(Model)
	if m.app.TargetLang != language.VI {
		t.Errorf("target = %q, want Vietnamese", m.app.TargetLang)
	}
	if got := m.input.Value(); got != "hello" {
		t.Errorf("draft = %q, want retained hello", got)
	}
}

func TestCtrlEFromProviderSavesKeyToJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TNOW_CONFIG", filepath.Join(dir, "config.yaml"))
	t.Setenv("TNOW_JSON_CONFIG", filepath.Join(dir, "config.json"))

	m := testModelWithProviders(t)
	m.openModelsBrowser()
	openaiIndex := -1
	for i, provider := range m.providerNames() {
		if provider == "openai" {
			openaiIndex = i
			break
		}
	}
	if openaiIndex < 0 {
		t.Fatal("openai provider not found")
	}
	m.mbProviderIdx = openaiIndex

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = tm.(Model)
	if !m.showAPIKey {
		t.Fatal("Ctrl+E should open the API key editor")
	}
	if m.pendingProvider != "openai" {
		t.Errorf("pending provider = %q, want openai", m.pendingProvider)
	}

	m.apiKeyInput.SetValue("sk-test-key")
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(Model)
	if m.showAPIKey {
		t.Fatal("API key editor should close after saving")
	}
	if !m.showModelsBrowser {
		t.Fatal("saving a key should return to the models browser")
	}
	if m.mbFocus != "model" {
		t.Errorf("browser focus = %q, want model", m.mbFocus)
	}
	if !strings.Contains(ansi.Strip(m.View()), "key saved") {
		t.Errorf("browser should show a key-saved notice:\n%s", ansi.Strip(m.View()))
	}

	saved, err := config.LoadJSON()
	if err != nil {
		t.Fatalf("load saved JSON config: %v", err)
	}
	provider, ok := saved.Provider("openai")
	if !ok {
		t.Fatal("openai provider missing from saved JSON config")
	}
	if provider.APIKey != "sk-test-key" {
		t.Errorf("saved API key = %q, want sk-test-key", provider.APIKey)
	}
}
