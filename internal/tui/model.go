package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/andev0x/tnow/internal/analysis"
	"github.com/andev0x/tnow/internal/app"
	"github.com/andev0x/tnow/internal/config"
	"github.com/andev0x/tnow/internal/language"
	"github.com/andev0x/tnow/internal/translation"
)

type Mode int

const (
	ModeQuick Mode = iota
	ModeAnalyze
)

type streamChunkMsg string
type streamErrMsg struct{ err error }
type streamDoneMsg struct{}
type analysisDoneMsg struct {
	res analysis.Result
	err error
}
type modelsListedMsg struct {
	provider string
	models   []string
	err      error
}

type slashCommand struct {
	Name        string
	Aliases     []string
	Usage       string
	Description string
}

var slashCommands = []slashCommand{
	{Name: "/to", Aliases: []string{"/lang", "/language", "/target"}, Usage: "/to <language|country>", Description: "set target language"},
	{Name: "/models", Aliases: []string{"/model", "/provider", "/providers"}, Usage: "/models [query]", Description: "browse providers & models"},
	{Name: "/word", Aliases: []string{"/analyze", "/mode"}, Usage: "/word [text]", Description: "toggle word analysis mode"},
	{Name: "/quit", Aliases: []string{"/exit", "/q"}, Usage: "/quit", Description: "exit application"},
	{Name: "/help", Aliases: []string{"/?"}, Usage: "/help", Description: "show available commands"},
}

func (c slashCommand) matches(prefix string) bool {
	if strings.HasPrefix(c.Name, prefix) {
		return true
	}
	for _, alias := range c.Aliases {
		if strings.HasPrefix(alias, prefix) {
			return true
		}
	}
	return false
}

// effectiveSlashLine returns the line that should be interpreted as a slash
// command. If the textarea contains multiline translation text plus a slash
// command on the last line (e.g. "hello\n/quick"), we treat the last
// non-empty line as the command so Quick mode can be activated without
// clearing the translation input first. For single-line input it is the
// trimmed value itself.
func effectiveSlashLine(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "\n") {
		return value
	}
	lines := strings.Split(value, "\n")
	// find last non-empty line
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return ""
}

// rawSlashLine returns the last non-empty line without trimming its trailing
// spaces. The trailing space matters: "/to " should open the language picker,
// while "/to" should still show the command itself.
func rawSlashLine(value string) string {
	if !strings.Contains(value, "\n") {
		return strings.TrimLeft(value, " \t")
	}
	lines := strings.Split(value, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return strings.TrimLeft(lines[i], " \t")
		}
	}
	return ""
}

// isLanguageCommand reports whether cmd is one of the /to aliases.
func isLanguageCommand(cmd string) bool {
	switch strings.ToLower(strings.TrimSpace(cmd)) {
	case "/to", "/lang", "/language", "/target":
		return true
	}
	return false
}

// slashLanguageQuery reports whether the active slash line is a language
// command with an argument area, e.g. "/to " or "/to vietnam".
func slashLanguageQuery(value string) (cmd, query string, ok bool) {
	line := rawSlashLine(value)
	if !strings.HasPrefix(line, "/") {
		return "", "", false
	}
	idx := strings.IndexAny(line, " \t")
	if idx < 0 {
		return "", "", false
	}
	cmd = strings.ToLower(strings.TrimSpace(line[:idx]))
	if !isLanguageCommand(cmd) {
		return "", "", false
	}
	return cmd, strings.TrimSpace(line[idx+1:]), true
}

// slashSuggestionsFor returns the commands matching the current slash input.
// It works on the effective slash line so that "hello\n/mo" still offers
// "/models" completion, and suggestions disappear once arguments start being typed.
func slashSuggestionsFor(value string) []slashCommand {
	line := effectiveSlashLine(value)
	if !strings.HasPrefix(line, "/") {
		return nil
	}
	// if the full value is multiline and the slash is not on the last line,
	// we already extracted last line, so no need to check Contains("\n")
	fields := strings.Fields(line)
	if len(fields) > 1 {
		return nil
	}
	prefix := strings.ToLower(fields[0])

	var matches []slashCommand
	for _, cmd := range slashCommands {
		if cmd.matches(prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// filterByPrefix keeps backward-compatible name but now supports search
// (case-insensitive substring matching) in addition to prefix matching.
func filterByPrefix(items []string, prefix string) []string {
	return filterBySearch(items, prefix)
}

// filterBySearch filters items by a case-insensitive search query. The query
// may contain multiple space-separated terms; an item matches when it contains
// every term. An empty query returns all items.
func filterBySearch(items []string, query string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return items
	}
	terms := strings.Fields(query)
	var filtered []string
	for _, item := range items {
		lower := strings.ToLower(item)
		ok := true
		for _, term := range terms {
			if !strings.Contains(lower, term) {
				ok = false
				break
			}
		}
		if ok {
			filtered = append(filtered, item)
		}
	}
	if filtered == nil {
		filtered = []string{}
	}
	return filtered
}

// Aliases for common naming variants used by search/filter helpers.
func filterByContains(items []string, query string) []string { return filterBySearch(items, query) }
func filterItems(items []string, query string) []string      { return filterBySearch(items, query) }
func searchItems(items []string, query string) []string      { return filterBySearch(items, query) }

// Exported aliases.
func FilterByPrefix(items []string, query string) []string   { return filterByPrefix(items, query) }
func FilterBySearch(items []string, query string) []string   { return filterBySearch(items, query) }
func FilterByContains(items []string, query string) []string { return filterByContains(items, query) }
func FilterItems(items []string, query string) []string      { return filterItems(items, query) }
func SearchItems(items []string, query string) []string      { return searchItems(items, query) }

type Model struct {
	app *app.App

	mode          Mode
	width, height int

	input          textarea.Model
	result         string
	analysisResult *analysis.Result
	status         string
	loading        bool

	spinner spinner.Model

	cancel context.CancelFunc
	chunks <-chan string
	errs   <-chan error

	// model overlay
	modelInput textinput.Model
	showModel  bool

	// highlight overlay & query
	highlightInput textinput.Model
	highlightQuery string
	showHighlight  bool

	// api key overlay (secure paste)
	apiKeyInput         textinput.Model
	showAPIKey          bool
	pendingProvider     string
	pendingModel        string
	apiKeyErr           string
	apiKeyReturnBrowser bool
	apiKeyFocus         string

	// languages
	langs   []language.Language
	langIdx int

	// provider/model cache
	modelCache map[string][]string
	modelErr   string

	// autocomplete
	suggestIdx      int
	filterExcellent bool

	// unified models browser (/models)
	showModelsBrowser bool
	mbFocus           string // "provider" | "model"
	mbProviderIdx     int
	mbModelIdx        int
	mbSearchInput     textinput.Model

	// result viewport scroll and panel focus
	resultScroll int
	focusOutput  bool

	styles Styles
}

func NewModel(a *app.App) Model {
	styles := DefaultStyles()

	ta := textarea.New()
	ta.Placeholder = "Type or paste text…"
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.SetHeight(4)

	// Paint every textarea surface explicitly. The screen compositor stains
	// transparent cells as a final safety net, but keeping the widget styles
	// opaque avoids mixed backgrounds around the cursor and wrapped lines.
	panelBG := lipgloss.NewStyle().Background(lipgloss.Color(ColorBgPanel))
	ta.FocusedStyle.Base = panelBG
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color(ColorBgPanel))
	ta.FocusedStyle.EndOfBuffer = panelBG
	ta.FocusedStyle.LineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextDim)).Background(lipgloss.Color(ColorBgPanel))
	ta.FocusedStyle.CursorLineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBorderFocused)).Background(lipgloss.Color(ColorBgPanel))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted)).Background(lipgloss.Color(ColorBgPanel))
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextPrimary)).Background(lipgloss.Color(ColorBgPanel))
	ta.BlurredStyle.Base = panelBG
	ta.BlurredStyle.CursorLine = panelBG
	ta.BlurredStyle.EndOfBuffer = panelBG
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextDim)).Background(lipgloss.Color(ColorBgPanel))
	ta.BlurredStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextSecondary)).Background(lipgloss.Color(ColorBgPanel))

	ti := textinput.New()
	ti.Placeholder = "e.g. gpt-4o, llama3.2:3b"
	ti.CharLimit = 128
	ti.Width = 42
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPrimary)).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextPrimary))
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted))

	hi := textinput.New()
	hi.Placeholder = "type word, phrase, or section to highlight..."
	hi.CharLimit = 64
	hi.Width = 42
	hi.Prompt = "Highlight: "
	hi.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorHighlightFg)).Bold(true)
	hi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorHighlightFg)).Bold(true)
	hi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted))

	ai := textinput.New()
	ai.Placeholder = "paste API key"
	ai.CharLimit = 256
	ai.Width = 42
	ai.EchoMode = textinput.EchoPassword
	ai.EchoCharacter = '•'
	ai.Prompt = "Key  "
	ai.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning)).Bold(true)
	ai.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextPrimary))
	ai.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted))

	si := textinput.New()
	si.Placeholder = "search models..."
	si.CharLimit = 64
	si.Prompt = "Search: "
	si.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPrimary)).Bold(true)
	si.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextPrimary))
	si.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted))

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning))

	langs := []language.Language{
		language.VI, language.EN, language.JA, language.KO,
		language.ZH, language.FR, language.ES, language.DE,
	}
	idx := 0
	for i, l := range langs {
		if l == a.TargetLang {
			idx = i
		}
	}

	return Model{
		app:             a,
		mode:            ModeQuick,
		input:           ta,
		modelInput:      ti,
		highlightInput:  hi,
		apiKeyInput:     ai,
		mbSearchInput:   si,
		spinner:         sp,
		langs:           langs,
		langIdx:         idx,
		modelCache:      make(map[string][]string),
		styles:          styles,
		suggestIdx:      -1,
		filterExcellent: false,
		mbFocus:         "provider",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m Model) cardWidth() int {
	w := m.width - 4
	if w > 100 {
		w = 100
	}
	if m.width > 0 && w > m.width-2 {
		w = m.width - 2
	}
	if w < 4 {
		w = 4
	}
	return w
}

// effectiveHeight returns a usable terminal height even before the first
// WindowSizeMsg arrives.
func (m Model) effectiveHeight() int {
	if m.height > 0 {
		return m.height
	}
	return 24
}

// layoutSizes returns card heights that depend only on the terminal height.
// Nothing about the draft, the slash palette, the model list, or the result
// can change these values, which is what keeps the screen from jumping while
// the user types.
//
// The main screen is assembled as:
//
//	header (2) + blank (1) + input (I) + blank (1) + output (O) + footer (1)
//
// so I+O == height-5 whenever the output panel is visible. The input takes
// roughly 42% of the body, clamped so the output always has room to read.
func (m Model) layoutSizes() (inputHeight, resultHeight int) {
	height := m.effectiveHeight()
	if height < 11 {
		// Too short for two useful panels: keep the editor and the footer.
		return max(2, height-4), 0
	}

	body := height - 5
	inputHeight = (body*42 + 50) / 100 // rounded 42%
	inputHeight = max(4, min(inputHeight, 14))
	if inputHeight > body-3 {
		inputHeight = body - 3
	}
	inputHeight = max(2, inputHeight)
	resultHeight = body - inputHeight
	if resultHeight < 3 {
		return max(2, height-4), 0
	}
	return inputHeight, resultHeight
}

// inputContentHeight is the number of text lines inside the input frame.
func (m Model) inputContentHeight() int {
	inputHeight, _ := m.layoutSizes()
	return max(1, inputHeight-2)
}

// syncInputSize keeps the real textarea viewport in sync with the fixed card.
// It is called from Update so cursor scrolling uses the same dimensions that
// View renders with, instead of changing the textarea only on a copy.
func (m Model) syncInputSize() Model {
	cardWidth := m.cardWidth()
	usableWidth := max(1, cardWidth-4)
	m.input.SetWidth(usableWidth)
	previousHeight := m.input.Height()
	m.input.SetHeight(m.inputContentHeight())
	if m.input.Height() != previousHeight {
		// SetHeight changes the viewport without scrolling it. A no-op key
		// update lets the textarea re-anchor the viewport around the cursor,
		// so a resize never hides the line being typed.
		m.input, _ = m.input.Update(tea.KeyMsg{Type: tea.KeyRunes})
	}
	return m
}

// suggestionItems returns the current suggestion list and its kind.
func (m Model) suggestionItems() (kind string, items []string, title, hint string) {
	if _, query, ok := slashLanguageQuery(m.input.Value()); ok {
		suggestions := m.languageSuggestions(query)
		if len(suggestions) > 0 {
			for _, suggestion := range suggestions {
				items = append(items, string(suggestion.Language))
			}
			return "language", items, "Languages", "↑↓ select · Tab accept · Enter apply"
		}
	}

	sugs := slashSuggestionsFor(m.input.Value())
	if len(sugs) == 0 {
		return "", nil, "", ""
	}
	title = "Commands"
	for _, s := range sugs {
		items = append(items, s.Name)
	}
	hint = "↑↓ navigate · Tab/Enter accept · Esc dismiss"
	return "command", items, title, hint
}

func (m Model) hasSuggestions() bool {
	kind, _, _, _ := m.suggestionItems()
	return kind != ""
}

// languageSuggestions keeps the CLI resolver order for real queries, but pins
// the active target language at the top of the unfiltered picker. That makes
// the current choice visible immediately in a compact popup.
func (m Model) languageSuggestions(query string) []language.Suggestion {
	suggestions := language.Suggest(query)
	if strings.TrimSpace(query) != "" || m.app.TargetLang == language.Auto {
		return suggestions
	}
	for i, suggestion := range suggestions {
		if suggestion.Language != m.app.TargetLang {
			continue
		}
		if i > 0 {
			pinned := suggestions[i]
			copy(suggestions[1:i+1], suggestions[:i])
			suggestions[0] = pinned
		}
		break
	}
	return suggestions
}

func splitInputForApply(value string) (retained string, slashLine string) {
	trimmed := strings.TrimSpace(value)
	if !strings.Contains(trimmed, "\n") {
		return "", trimmed
	}
	lines := strings.Split(trimmed, "\n")
	// find last non-empty line as slash line
	lastIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastIdx = i
			break
		}
	}
	if lastIdx == -1 {
		return "", trimmed
	}
	slashLine = strings.TrimSpace(lines[lastIdx])
	if !strings.HasPrefix(slashLine, "/") {
		return "", trimmed
	}
	retained = strings.TrimSpace(strings.Join(lines[:lastIdx], "\n"))
	return retained, slashLine
}

func (m *Model) applySuggestion(choice string) {
	val := m.input.Value()
	retained, slashLine := splitInputForApply(val)
	line := slashLine
	if line == "" {
		line = strings.TrimSpace(val)
	}
	if strings.Contains(strings.TrimSpace(val), "\n") && retained == "" && !strings.HasPrefix(strings.TrimSpace(val), "/") {
		return
	}
	kind, _, _, _ := m.suggestionItems()
	switch kind {
	case "command":
		newLine := choice + " "
		if retained != "" {
			m.input.SetValue(retained + "\n" + newLine)
		} else {
			m.input.SetValue(newLine)
		}
		m.suggestIdx = -1
	case "provider", "model":
		parts := strings.SplitN(line, " ", 2)
		cmd := parts[0]
		newLine := cmd + " " + choice
		if retained != "" {
			m.input.SetValue(retained + "\n" + newLine)
		} else {
			m.input.SetValue(newLine)
		}
		m.suggestIdx = -1
	case "language":
		fields := strings.Fields(line)
		cmd := "/to"
		if len(fields) > 0 {
			cmd = fields[0]
		}
		newLine := cmd + " " + choice
		if retained != "" {
			m.input.SetValue(retained + "\n" + newLine)
		} else {
			m.input.SetValue(newLine)
		}
		m.suggestIdx = -1
	}
}

func (m *Model) clampSuggestIdx(count int) {
	if count == 0 {
		m.suggestIdx = -1
		return
	}
	if m.suggestIdx < -1 {
		m.suggestIdx = count - 1
	}
	if m.suggestIdx >= count {
		m.suggestIdx = 0
	}
}

// openAPIKey opens the compact API-key editor for a provider. The models
// browser, when open, stays in the background so the user returns to the same
// provider/model context after saving or cancelling.
func (m *Model) openAPIKey(provider string, pendingModel string) {
	if provider == "" {
		return
	}
	m.pendingProvider = provider
	m.pendingModel = pendingModel
	m.apiKeyReturnBrowser = m.showModelsBrowser
	m.apiKeyFocus = m.mbFocus
	m.showAPIKey = true
	m.apiKeyInput.Width = max(8, min(54, m.width-24))
	m.apiKeyInput.SetValue("")
	m.apiKeyInput.Focus()
	m.apiKeyErr = ""
	m.input.Blur()
	m.mbSearchInput.Blur()
	m.status = "enter an API key for " + provider
}

func (m *Model) closeAPIKey() {
	m.showAPIKey = false
	m.apiKeyInput.SetValue("")
	m.apiKeyInput.Blur()
	m.apiKeyErr = ""
	m.pendingProvider = ""
	m.pendingModel = ""
	if m.apiKeyReturnBrowser && m.showModelsBrowser {
		m.mbSearchInput.Focus()
		if m.apiKeyFocus == "provider" || m.apiKeyFocus == "model" {
			m.mbFocus = m.apiKeyFocus
		}
	} else {
		m.input.Focus()
	}
	m.apiKeyReturnBrowser = false
	m.apiKeyFocus = ""
}

// modelIndexForProvider returns the index of the active model for the active
// provider, or zero for any other provider.
func (m Model) modelIndexForProvider(provider string) int {
	if provider != m.app.Provider || m.app.Model == "" {
		return 0
	}
	models := m.mbFilteredModels(provider)
	for i, model := range models {
		if model == m.app.Model {
			return i
		}
	}
	return 0
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.syncInputSize()
		cardWidth := m.cardWidth()
		searchWidth := cardWidth - 10
		if searchWidth < 8 {
			searchWidth = 8
		}
		m.mbSearchInput.Width = searchWidth
		if m.showAPIKey {
			m.apiKeyInput.Width = max(8, min(54, m.width-24))
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Keep the editor viewport at the fixed card size before processing a
		// key so wrapped lines scroll to the cursor instead of changing layout.
		m = m.syncInputSize()

		// API key overlay has highest priority
		if m.showAPIKey {
			switch msg.String() {
			case "ctrl+d":
				if m.cancel != nil {
					m.cancel()
				}
				return m, tea.Quit
			case "ctrl+c", "esc":
				m.closeAPIKey()
				m.status = "cancelled"
				return m, nil
			case "enter":
				key := strings.TrimSpace(m.apiKeyInput.Value())
				if key == "" {
					m.apiKeyErr = "API key cannot be empty"
					return m, nil
				}
				prov := m.pendingProvider
				if err := m.app.SetProviderAPIKey(prov, key); err != nil {
					m.apiKeyErr = err.Error()
					return m, nil
				}
				pendingModel := m.pendingModel
				m.showAPIKey = false
				m.apiKeyInput.SetValue("")
				m.apiKeyErr = ""
				m.pendingProvider = ""
				m.pendingModel = ""
				m.app.SetProvider(prov)
				if pendingModel != "" {
					m.app.SetModel(pendingModel)
				}
				m.status = "key saved for " + prov + " · " + prettyConfigPath()
				if m.apiKeyReturnBrowser && m.showModelsBrowser {
					m.apiKeyReturnBrowser = false
					m.apiKeyFocus = ""
					m.mbSearchInput.Focus()
					m.mbFocus = "model"
					m.mbModelIdx = m.modelIndexForProvider(prov)
					return m, m.fetchModelsCmd(prov)
				}
				m.apiKeyReturnBrowser = false
				m.apiKeyFocus = ""
				m.input.Focus()
				return m, m.fetchModelsCmd(prov)
			}
			var cmd tea.Cmd
			m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
			return m, cmd
		}

		// Model overlay
		if m.showModel {
			switch msg.String() {
			case "ctrl+d":
				if m.cancel != nil {
					m.cancel()
				}
				return m, tea.Quit
			case "ctrl+c":
				m.showModel = false
				m.modelInput.SetValue("")
				return m, nil
			case "enter":
				val := strings.TrimSpace(m.modelInput.Value())
				if val != "" {
					m.app.SetModel(val)
				}
				m.showModel = false
				m.modelInput.SetValue("")
				return m, nil
			case "esc":
				m.showModel = false
				m.modelInput.SetValue("")
				return m, nil
			}
			var cmd tea.Cmd
			m.modelInput, cmd = m.modelInput.Update(msg)
			return m, cmd
		}

		// Highlight overlay
		if m.showHighlight {
			switch msg.String() {
			case "ctrl+d":
				if m.cancel != nil {
					m.cancel()
				}
				return m, tea.Quit
			case "ctrl+c":
				m.showHighlight = false
				return m, nil
			case "enter":
				m.highlightQuery = strings.TrimSpace(m.highlightInput.Value())
				m.showHighlight = false
				return m, nil
			case "esc":
				m.showHighlight = false
				return m, nil
			}
			var cmd tea.Cmd
			m.highlightInput, cmd = m.highlightInput.Update(msg)
			return m, cmd
		}

		// Unified models browser (/models)
		if m.showModelsBrowser {
			providers := m.providerNames()
			if len(providers) == 0 {
				if msg.String() == "esc" || msg.String() == "ctrl+c" {
					m.showModelsBrowser = false
					m.mbSearchInput.Blur()
					m.input.Focus()
				}
				return m, nil
			}
			if m.mbProviderIdx < 0 {
				m.mbProviderIdx = 0
			}
			if m.mbProviderIdx >= len(providers) {
				m.mbProviderIdx = len(providers) - 1
			}
			selectedProv := providers[m.mbProviderIdx]
			models := m.mbFilteredModels(selectedProv)
			if m.mbModelIdx < 0 || m.mbModelIdx >= len(models) {
				m.mbModelIdx = 0
			}

			switch msg.String() {
			case "esc":
				m.showModelsBrowser = false
				m.mbSearchInput.SetValue("")
				m.mbSearchInput.Blur()
				m.input.Focus()
				m.status = ""
				return m, nil
			case "ctrl+d":
				if m.cancel != nil {
					m.cancel()
				}
				return m, tea.Quit
			case "ctrl+c":
				// Clear the model search without closing the browser.
				m.mbSearchInput.SetValue("")
				m.mbModelIdx = m.modelIndexForProvider(selectedProv)
				return m, nil
			case "left", "right":
				if m.mbFocus == "provider" {
					m.mbFocus = "model"
					m.mbModelIdx = m.modelIndexForProvider(selectedProv)
					if len(models) == 0 {
						return m, m.fetchModelsCmd(selectedProv)
					}
				} else {
					m.mbFocus = "provider"
				}
				return m, nil
			case "up", "ctrl+p":
				if m.mbFocus == "provider" {
					m.mbProviderIdx--
					if m.mbProviderIdx < 0 {
						m.mbProviderIdx = len(providers) - 1
					}
					selectedProv = providers[m.mbProviderIdx]
					m.mbModelIdx = m.modelIndexForProvider(selectedProv)
					return m, m.fetchModelsCmd(selectedProv)
				}
				if len(models) > 0 {
					m.mbModelIdx--
					if m.mbModelIdx < 0 {
						m.mbModelIdx = len(models) - 1
					}
				}
				return m, nil
			case "down", "ctrl+n":
				if m.mbFocus == "provider" {
					m.mbProviderIdx++
					if m.mbProviderIdx >= len(providers) {
						m.mbProviderIdx = 0
					}
					selectedProv = providers[m.mbProviderIdx]
					m.mbModelIdx = m.modelIndexForProvider(selectedProv)
					return m, m.fetchModelsCmd(selectedProv)
				}
				if len(models) > 0 {
					m.mbModelIdx++
					if m.mbModelIdx >= len(models) {
						m.mbModelIdx = 0
					}
				}
				return m, nil
			case "enter":
				if m.mbFocus == "provider" {
					if config.ProviderRequiresKey(selectedProv) && !m.app.HasAPIKey(selectedProv) {
						m.openAPIKey(selectedProv, "")
						m.status = "add an API key for " + selectedProv + " · Ctrl+E"
						return m, nil
					}
					m.mbFocus = "model"
					m.mbModelIdx = m.modelIndexForProvider(selectedProv)
					return m, m.fetchModelsCmd(selectedProv)
				}
				if len(models) == 0 {
					return m, m.fetchModelsCmd(selectedProv)
				}
				if len(models) > 0 && m.mbModelIdx >= 0 && m.mbModelIdx < len(models) {
					chosenModel := models[m.mbModelIdx]
					chosenProvider := selectedProv
					if config.ProviderRequiresKey(chosenProvider) && !m.app.HasAPIKey(chosenProvider) {
						m.openAPIKey(chosenProvider, chosenModel)
						m.status = "add an API key for " + chosenProvider + " to use " + chosenModel
						return m, nil
					}
					m.app.SetProvider(chosenProvider)
					m.app.SetModel(chosenModel)
					m.mbSearchInput.SetValue("")
					m.mbModelIdx = 0
					m.status = "provider " + chosenProvider + " · model " + chosenModel
					return m, m.fetchModelsCmd(chosenProvider)
				}
				return m, nil
			case "ctrl+t":
				m.filterExcellent = !m.filterExcellent
				m.mbModelIdx = m.modelIndexForProvider(selectedProv)
				if m.filterExcellent {
					m.status = "filtering translation-optimized models"
				} else {
					m.status = "showing all models"
				}
				return m, nil
			case "ctrl+e":
				m.openAPIKey(selectedProv, "")
				m.status = "enter an API key for " + selectedProv + " · " + prettyConfigPath()
				return m, nil
			}
			// All other keys edit the model search query.
			prev := m.mbSearchInput.Value()
			var cmd tea.Cmd
			m.mbSearchInput, cmd = m.mbSearchInput.Update(msg)
			if m.mbSearchInput.Value() != prev {
				m.mbModelIdx = m.modelIndexForProvider(selectedProv)
			}
			return m, cmd
		}

		// Suggestion navigation when suggestions visible.
		if m.hasSuggestions() {
			kind, items, _, _ := m.suggestionItems()
			switch msg.String() {
			case "up", "ctrl+p":
				if len(items) > 0 {
					if m.suggestIdx == -1 {
						m.suggestIdx = len(items) - 1
					} else {
						m.suggestIdx--
						m.clampSuggestIdx(len(items))
					}
				}
				return m, nil
			case "down", "ctrl+n":
				if len(items) > 0 {
					if m.suggestIdx == -1 {
						m.suggestIdx = 0
					} else {
						m.suggestIdx++
						m.clampSuggestIdx(len(items))
					}
				}
				return m, nil
			case "tab":
				if len(items) > 0 {
					idx := m.suggestIdx
					if idx < 0 || idx >= len(items) {
						idx = 0
					}
					m.applySuggestion(items[idx])
					m.suggestIdx = -1
				}
				return m, nil
			case "esc":
				if m.suggestIdx != -1 {
					m.suggestIdx = -1
					return m, nil
				}
				// With no selection, let the normal Esc handling clear a
				// highlight or cancel an in-flight request.
			}
			if msg.String() == "enter" {
				if kind == "language" {
					selected := -1
					if m.suggestIdx >= 0 && m.suggestIdx < len(items) {
						selected = m.suggestIdx
					}
					if selected >= 0 {
						m.applySuggestion(items[selected])
						m.suggestIdx = -1
						return m.submit()
					}
					_, query, _ := slashLanguageQuery(m.input.Value())
					if _, ok := language.Resolve(query); ok {
						return m.submit()
					}
					m.status = "choose a language · ↑↓ then Enter"
					return m, nil
				}

				line := strings.ToLower(effectiveSlashLine(m.input.Value()))
				if isLanguageCommand(line) {
					m.applySuggestion(line)
					m.suggestIdx = -1
					return m, nil
				}
				for _, command := range slashCommands {
					if line == strings.ToLower(command.Name) {
						return m.submit()
					}
					for _, alias := range command.Aliases {
						if line == strings.ToLower(alias) {
							return m.submit()
						}
					}
				}
				if m.suggestIdx >= 0 && m.suggestIdx < len(items) {
					m.applySuggestion(items[m.suggestIdx])
					m.suggestIdx = -1
					return m, nil
				}
				if len(items) == 1 {
					m.applySuggestion(items[0])
					m.suggestIdx = -1
					return m, nil
				}
				// Keep the menu open while the user chooses a command.
				return m, nil
			}
		}

		// Main keybindings
		switch msg.String() {
		case "ctrl+d":
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit

		case "ctrl+c":
			// delete content input (clear textarea and result)
			if m.loading && m.cancel != nil {
				m.cancel()
				m.loading = false
				m.status = "cancelled"
			}
			m.input.SetValue("")
			m.result = ""
			m.analysisResult = nil
			m.resultScroll = 0
			m.focusOutput = false
			m.status = ""
			m.input.Focus()
			return m, nil

		case "esc":
			if m.highlightQuery != "" {
				m.highlightQuery = ""
				m.highlightInput.SetValue("")
				return m, nil
			}
			if m.hasSuggestions() {
				// Close the palette without discarding a translation draft
				// that sits on the lines above the slash command.
				retained, _ := splitInputForApply(m.input.Value())
				m.input.SetValue(retained)
				m.input.CursorEnd()
				m.suggestIdx = -1
				m.status = ""
				m = m.syncInputSize()
				return m, nil
			}
			if m.loading && m.cancel != nil {
				m.cancel()
				m.loading = false
				m.status = "cancelled"
				return m, nil
			}
			if m.focusOutput {
				m.focusOutput = false
				return m, nil
			}
			return m, nil

		case "tab":
			if m.mode == ModeQuick {
				m.mode = ModeAnalyze
				m.input.Placeholder = "Type a word or phrase…"
			} else {
				m.mode = ModeQuick
				m.input.Placeholder = "Type or paste text…"
			}
			m.result = ""
			m.analysisResult = nil
			m.resultScroll = 0
			m.focusOutput = false
			m.status = ""
			m.suggestIdx = -1
			return m, nil

		case "ctrl+l":
			m.langIdx = (m.langIdx + 1) % len(m.langs)
			m.app.SetTarget(m.langs[m.langIdx])
			return m, nil

		case "ctrl+m", "alt+m":
			m.showModel = true
			m.modelInput.SetValue(m.app.Model)
			m.modelInput.Focus()
			return m, nil

		case "ctrl+h":
			m.showHighlight = true
			m.highlightInput.SetValue(m.highlightQuery)
			m.highlightInput.Focus()
			return m, nil

		case "ctrl+e":
			prov := m.app.Provider
			if prov == "" {
				provs := m.providerNames()
				if len(provs) > 0 {
					prov = provs[0]
				}
			}
			if prov != "" {
				m.openAPIKey(prov, "")
				m.status = "enter an API key for " + prov + " · " + prettyConfigPath()
			}
			return m, nil

		case "ctrl+y":
			if m.result != "" {
				_ = clipboard.WriteAll(m.result)
				m.status = "copied"
			}
			return m, nil

		case "pgup":
			m.scrollResult(-m.resultPageStep())
			return m, nil
		case "pgdown":
			m.scrollResult(m.resultPageStep())
			return m, nil
		case "ctrl+u", "alt+up":
			m.scrollResult(-1)
			return m, nil
		case "alt+down", "ctrl+down":
			m.scrollResult(1)
			return m, nil

		case "ctrl+s", "ctrl+enter", "alt+enter", "ctrl+j":
			return m.submit()
		case "enter":
			// Suggestion Enter handling already ran above. By the time we get
			// here, a slash command with arguments (or an unknown command) can
			// be submitted; ordinary Enter inserts a newline in the textarea.
			if strings.HasPrefix(effectiveSlashLine(m.input.Value()), "/") {
				return m.submit()
			}
		}

	case streamChunkMsg:
		m.result += string(msg)
		if m.status == "translating..." {
			m.status = "streaming..."
		}
		if m.chunks != nil {
			return m, waitForChunk(m.chunks, m.errs)
		}
		return m, nil

	case streamErrMsg:
		if !m.loading {
			return m, nil
		}
		m.loading = false
		m.focusOutput = false
		m.status = msg.err.Error()
		m.cancel = nil
		return m, nil

	case streamDoneMsg:
		if !m.loading {
			return m, nil
		}
		m.loading = false
		m.focusOutput = true
		m.cancel = nil
		m.status = "done"
		m.resultScroll = 0
		// auto copy selected output translation
		if m.app.Config.AutoCopy && strings.TrimSpace(m.result) != "" {
			_ = clipboard.WriteAll(strings.TrimSpace(m.result))
			m.status = "auto-copied ✓"
		}
		return m, nil

	case analysisDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.focusOutput = false
			m.status = msg.err.Error()
			return m, nil
		}
		m.focusOutput = true
		m.analysisResult = &msg.res
		m.result = formatAnalysis(msg.res)
		m.status = "done"
		m.resultScroll = 0
		if m.app.Config.AutoCopy && strings.TrimSpace(m.result) != "" {
			_ = clipboard.WriteAll(strings.TrimSpace(m.result))
			m.status = "auto-copied ✓"
		}
		return m, nil

	case modelsListedMsg:
		selectedModel := ""
		if m.showModelsBrowser {
			providers := m.providerNames()
			if m.mbProviderIdx >= 0 && m.mbProviderIdx < len(providers) && providers[m.mbProviderIdx] == msg.provider {
				models := m.mbFilteredModels(msg.provider)
				if m.mbModelIdx >= 0 && m.mbModelIdx < len(models) {
					selectedModel = models[m.mbModelIdx]
				}
			}
		}
		if msg.err != nil {
			m.modelErr = strings.ReplaceAll(msg.err.Error(), "\n", " ")
		} else {
			m.modelErr = ""
			if len(msg.models) > 0 {
				if m.modelCache == nil {
					m.modelCache = make(map[string][]string)
				}
				m.modelCache[msg.provider] = msg.models
			}
		}
		if m.showModelsBrowser {
			providers := m.providerNames()
			if m.mbProviderIdx >= 0 && m.mbProviderIdx < len(providers) && providers[m.mbProviderIdx] == msg.provider {
				models := m.mbFilteredModels(msg.provider)
				if selectedModel != "" {
					for i, model := range models {
						if model == selectedModel {
							m.mbModelIdx = i
							return m, nil
						}
					}
				}
				if m.mbModelIdx < 0 || m.mbModelIdx >= len(models) {
					m.mbModelIdx = 0
				}
			}
		}
		return m, nil
	}

	// default update input
	var cmd tea.Cmd
	prevVal := m.input.Value()
	m.input, cmd = m.input.Update(msg)
	// Typing changes the suggestion set, so an old highlight would point at a
	// different row. Keep navigation state only for up/down, which return
	// before reaching this path.
	if m.input.Value() != prevVal {
		m.suggestIdx = -1
	}
	// Any editor key returns the visual focus to the input panel.
	m.focusOutput = false
	m = m.syncInputSize()
	return m, cmd
}

func (m Model) submit() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		return m, nil
	}
	// Slash commands are local state changes, so they should still work
	// while a previous translation/analysis request is in flight.
	if strings.HasPrefix(effectiveSlashLine(text), "/") {
		return m.handleSlashCommand(text)
	}
	if m.loading {
		return m, nil
	}
	m.suggestIdx = -1
	m.result = ""
	m.analysisResult = nil
	m.resultScroll = 0
	m.focusOutput = true
	m.status = "translating..."
	m.loading = true

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	if m.mode == ModeQuick {
		req := translation.Request{
			Text:       text,
			SourceLang: m.app.SourceLang,
			TargetLang: m.app.TargetLang,
		}
		outCh, errCh, _, _, err := m.app.Translation.TranslateStream(ctx, req)
		if err != nil {
			m.loading = false
			m.focusOutput = false
			m.status = err.Error()
			cancel()
			m.cancel = nil
			return m, nil
		}
		m.chunks = outCh
		m.errs = errCh
		return m, waitForChunk(outCh, errCh)
	}

	// analyze mode
	m.status = "analyzing..."
	return m, func() tea.Msg {
		res, err := m.app.Analysis.Analyze(ctx, analysis.Request{
			Word:       text,
			SourceLang: m.app.SourceLang,
			TargetLang: m.app.TargetLang,
		})
		return analysisDoneMsg{res: res, err: err}
	}
}

func (m Model) fetchModelsCmd(provider string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		models, err := m.app.Runtime.ListModels(ctx, provider)
		return modelsListedMsg{provider: provider, models: models, err: err}
	}
}

func (m Model) providerNames() []string {
	return m.app.ProviderNames()
}

func (m Model) FilterByPrefix(items []string, query string) []string {
	return filterByPrefix(items, query)
}
func (m Model) FilterBySearch(items []string, query string) []string {
	return filterBySearch(items, query)
}
func (m Model) FilterByContains(items []string, query string) []string {
	return filterByContains(items, query)
}

func (m Model) SearchModels(args ...string) []string {
	if len(args) == 0 {
		return m.searchModels("")
	}
	if len(args) == 1 {
		return m.searchModels(args[0])
	}
	return m.searchModelsForProvider(args[0], args[1])
}
func (m Model) SearchModelsForProvider(args ...string) []string {
	if len(args) == 0 {
		return m.searchModelsForProvider(m.app.Provider, "")
	}
	if len(args) == 1 {
		// ambiguous: treat as query for active provider
		return m.searchModelsForProvider(m.app.Provider, args[0])
	}
	return m.searchModelsForProvider(args[0], args[1])
}
func (m Model) SearchProviders(args ...string) []string {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}
	return m.searchProviders(query)
}

func (m Model) searchModels(query string) []string {
	return filterBySearch(m.modelsForProvider(m.app.Provider), query)
}

func (m Model) searchModelsForProvider(name, query string) []string {
	return filterBySearch(m.modelsForProvider(name), query)
}

func (m Model) searchProviders(query string) []string {
	return filterBySearch(m.providerNames(), query)
}

func (m *Model) mbFilteredModels(provider string) []string {
	all := m.modelsForProvider(provider)
	activeModel := ""
	if provider == m.app.Provider {
		activeModel = m.app.Model
	}

	// The search query applies to models. The active model is always kept when
	// it matches, but never injected into an unrelated search result.
	if query := strings.TrimSpace(m.mbSearchInput.Value()); query != "" {
		all = filterBySearch(all, query)
	}

	if m.filterExcellent {
		filtered := make([]string, 0, len(all))
		for _, model := range all {
			if model == activeModel || app.IsTranslationExcellent(model) {
				filtered = append(filtered, model)
			}
		}
		all = filtered
	}

	// The active model is the most useful "appropriate" choice, so pin it at
	// the top. Translation-oriented models follow, then alphabetical order.
	sort.SliceStable(all, func(i, j int) bool {
		if all[i] == activeModel && all[j] != activeModel {
			return true
		}
		if all[j] == activeModel && all[i] != activeModel {
			return false
		}
		leftExcellent := app.IsTranslationExcellent(all[i])
		rightExcellent := app.IsTranslationExcellent(all[j])
		if leftExcellent != rightExcellent {
			return leftExcellent
		}
		return all[i] < all[j]
	})

	if all == nil {
		return []string{}
	}
	return all
}

func (m *Model) openModelsBrowser() tea.Cmd {
	providers := m.providerNames()
	idx := 0
	for i, p := range providers {
		if p == m.app.Provider {
			idx = i
			break
		}
	}
	m.showModelsBrowser = true
	m.mbFocus = "provider"
	m.mbProviderIdx = idx
	m.mbSearchInput.SetValue("")
	m.modelErr = ""
	m.mbModelIdx = m.modelIndexForProvider(providers[idx])
	m.mbSearchInput.Focus()
	m.input.Blur()
	m.status = "↑↓ navigate · ←→ panes · Enter assign · Ctrl+E key · Esc close"
	if len(providers) > 0 {
		return m.fetchModelsCmd(providers[idx])
	}
	return nil
}

func (m Model) modelsForProvider(name string) []string {
	seen := make(map[string]bool)
	var models []string
	add := func(model string) {
		if model == "" || seen[model] {
			return
		}
		seen[model] = true
		models = append(models, model)
	}
	for _, model := range m.modelCache[name] {
		add(model)
	}
	for _, model := range m.app.ModelsForProvider(name) {
		add(model)
	}
	sort.Strings(models)
	return models
}

func (m Model) handleSlashCommand(raw string) (tea.Model, tea.Cmd) {
	// Support multiline input where translation text is on previous lines
	// and the slash command is on the last line, e.g. "hello\n/quit".
	// We extract the command line and retain the preceding translation text
	// so that switching modes does not discard the user's input.
	retained, cmdLine := splitInputForApply(raw)
	if cmdLine == "" {
		cmdLine = strings.TrimSpace(raw)
		retained = ""
	} else if retained == "" && !strings.HasPrefix(strings.TrimSpace(raw), "/") {
		// splitInputForApply already handled multiline; for single-line it returns "".
		// Ensure we still parse the single-line command.
		cmdLine = strings.TrimSpace(raw)
		if strings.HasPrefix(cmdLine, "/") && strings.Contains(raw, "\n") {
			// multiline case already extracted retained correctly
		} else {
			retained = ""
		}
	}
	// If raw was multiline but cmdLine extraction failed, fallback to effective line
	if !strings.HasPrefix(cmdLine, "/") {
		cmdLine = effectiveSlashLine(raw)
		if cmdLine == "" {
			return m, nil
		}
		// re-extract retained for this fallback
		r, c := splitInputForApply(raw)
		if c != "" {
			retained, cmdLine = r, c
		}
	}
	fields := strings.Fields(cmdLine)
	if len(fields) == 0 {
		return m, nil
	}
	cmd := strings.ToLower(fields[0])
	args := fields[1:]
	m.suggestIdx = -1
	// helper to restore retained translation text after mode switches
	restoreRetained := func() {
		if retained != "" {
			m.input.SetValue(retained)
		} else {
			m.input.SetValue("")
		}
	}

	switch cmd {
	case "/to", "/lang", "/language", "/target":
		if len(args) == 0 {
			m.status = "usage: /to <language or country>"
			restoreRetained()
			return m, nil
		}
		target := strings.Trim(strings.Join(args, " "), `"'`)
		lang, ok := language.Resolve(target)
		if !ok {
			message := "unknown language or country: " + target
			if suggestions := language.Suggest(target); len(suggestions) > 0 {
				best := suggestions[0]
				message += " · try " + string(best.Language) + " (" + best.Name + ")"
			}
			m.status = message
			restoreRetained()
			return m, nil
		}
		if lang == language.Auto {
			m.status = "target cannot be auto · source detection is automatic"
			restoreRetained()
			return m, nil
		}
		m.app.SetTarget(lang)
		for i, l := range m.langs {
			if l == lang {
				m.langIdx = i
			}
		}
		m.status = "target language set to " + language.DisplayName(lang) + " (" + string(lang) + ")"

	case "/models", "/model":
		// unified provider+model browser, optionally pre-filtered
		restoreRetained()
		browserCmd := m.openModelsBrowser()
		if len(args) > 0 {
			m.mbSearchInput.SetValue(strings.Join(args, " "))
			m.mbModelIdx = 0
		}
		return m, browserCmd

	case "/provider", "/providers":
		restoreRetained()
		browserCmd := m.openModelsBrowser()
		if len(args) > 0 {
			query := strings.ToLower(strings.Join(args, " "))
			for i, provider := range m.providerNames() {
				name := strings.ToLower(provider)
				if strings.HasPrefix(name, query) || strings.Contains(name, query) {
					m.mbProviderIdx = i
					m.mbFocus = "provider"
					m.mbModelIdx = m.modelIndexForProvider(provider)
					browserCmd = m.fetchModelsCmd(provider)
					break
				}
			}
		}
		return m, browserCmd

	case "/word", "/analyze", "/mode":
		if len(args) > 0 {
			m.mode = ModeAnalyze
			// if there was retained translation, args take precedence
			m.input.SetValue(strings.Join(args, " "))
			m.input.Placeholder = "Type a word or phrase…"
			m.status = "word analysis mode"
		} else {
			if m.mode == ModeQuick {
				m.mode = ModeAnalyze
				m.input.Placeholder = "Type a word or phrase…"
				m.status = "word analysis mode"
			} else {
				m.mode = ModeQuick
				m.input.Placeholder = "Type or paste text…"
				m.status = "quick translation mode"
			}
			restoreRetained()
		}
		m.result = ""
		m.analysisResult = nil

	case "/quit", "/exit", "/q":
		return m, tea.Quit

	case "/help", "/?":
		m.status = "commands: /to <language|country>  /models [query]  /word [text]  /help  · Ctrl+E add API key"
		restoreRetained()

	default:
		m.status = "unknown command: " + cmd
		restoreRetained()
		return m, nil
	}

	// clear input for commands that did not explicitly set it
	// keep translation text (retained) when switching modes/languages so quick
	// mode stays active with the user's input intact
	if cmd != "/word" || len(args) == 0 {
		if retained != "" {
			// preserve translation text; e.g. "hello\n/to vi" -> keep "hello"
			if m.input.Value() != retained {
				m.input.SetValue(retained)
			}
		} else {
			// single-line slash command: input should be cleared (already done
			// via restoreRetained in most branches)
			if strings.HasPrefix(effectiveSlashLine(m.input.Value()), "/") {
				m.input.SetValue("")
			}
		}
	}
	return m, nil
}

func waitForChunk(ch <-chan string, errCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case c, ok := <-ch:
			if !ok {
				select {
				case e := <-errCh:
					if e != nil {
						return streamErrMsg{err: e}
					}
				default:
				}
				return streamDoneMsg{}
			}
			return streamChunkMsg(c)
		case e := <-errCh:
			if e != nil {
				return streamErrMsg{err: e}
			}
			return streamDoneMsg{}
		}
	}
}

func (m Model) View() string {
	width := m.width
	if width == 0 {
		width = 80
	}
	height := m.effectiveHeight()
	cardWidth := m.cardWidth()
	cardX := max(0, (width-cardWidth)/2)

	// Overlays take priority. They are painted on the same opaque canvas as
	// the main screen so dialogs never inherit terminal or panel artifacts.
	if m.showAPIKey {
		return m.renderAPIKeyModal()
	}
	if m.showModel {
		m.modelInput.Width = max(12, min(54, m.width-24))
		return m.renderModal(
			"Configure Model",
			"Enter model identifier:",
			m.modelInput.View(),
			"Enter to save  ·  Esc to cancel",
		)
	}
	if m.showHighlight {
		m.highlightInput.Width = max(12, min(54, m.width-24))
		return m.renderModal(
			"Highlight Sections / Terms",
			"Enter a word or pattern to highlight:",
			m.highlightInput.View(),
			"Enter to apply  ·  Esc to dismiss",
		)
	}
	if m.showModelsBrowser {
		return m.renderModelsBrowser()
	}

	inputHeight, resultHeight := m.layoutSizes()
	inputY := 3
	outputY := inputY + inputHeight + 1
	footerY := max(0, height-1)

	layers := []screenLayer{
		{x: cardX, y: 0, width: cardWidth, height: 2, text: m.renderHeader(cardWidth), bg: ColorBgBase},
		{x: cardX, y: inputY, width: cardWidth, height: inputHeight, text: m.renderInputCard(cardWidth, inputHeight), bg: ColorBgPanel},
	}
	if resultHeight > 0 {
		layers = append(layers, screenLayer{
			x: cardX, y: outputY, width: cardWidth, height: resultHeight,
			text: m.renderResultCard(cardWidth, resultHeight), bg: ColorBgPanel,
		})
	}
	layers = append(layers, screenLayer{
		x: cardX, y: footerY, width: cardWidth, height: 1,
		text: m.renderFooter(cardWidth), bg: ColorBgBar,
	})

	// The slash palette is a true floating overlay. It never changes the base
	// card heights; filtering only redraws inside the same fixed popup.
	if m.hasSuggestions() && cardWidth >= 14 {
		popupWidth, popupHeight := 0, 0
		var popupX, popupY int
		withShadow := false

		switch {
		case resultHeight >= 6:
			// Normal panes: float inside the output panel, leaving its title
			// and bottom border visible around the dropdown.
			available := resultHeight - 2
			popupWidth = min(cardWidth-4, 78)
			if popupWidth < 12 {
				popupWidth = max(4, cardWidth-2)
			}
			popupHeight = min(7, available)
			popupX = cardX + (cardWidth-popupWidth)/2
			popupY = outputY + 1
			withShadow = true
		case resultHeight > 0:
			// Very short panes: use the output slot as the popup surface.
			// This avoids overlapping the input or leaving border fragments
			// from a panel too small to contain an inset dialog.
			popupWidth = cardWidth
			popupHeight = resultHeight
			popupX = cardX
			popupY = outputY
		default:
			// No output panel: float over the lower part of the editor while
			// keeping the panel title and the line being typed visible.
			popupWidth = cardWidth
			popupHeight = min(inputHeight, min(5, max(3, inputHeight-2)))
			popupX = cardX
			popupY = inputY + inputHeight - popupHeight
		}

		if popupWidth >= 8 && popupHeight >= 3 {
			popup := m.renderCommandPopup(popupWidth, popupHeight)
			if withShadow {
				layers = append(layers, screenLayer{
					x: popupX + 1, y: popupY + 1, width: popupWidth,
					height: max(1, popupHeight-1), bg: ColorBgShadow,
				})
			}
			layers = append(layers, screenLayer{
				x: popupX, y: popupY, width: popupWidth,
				height: popupHeight, text: popup, bg: ColorBgRaised,
			})
		}
	}

	return renderScreen(width, height, ColorBgBase, layers...)
}

// langPair is the compact language indicator used by the header and output
// panel metadata.
func (m Model) langPair() string {
	src := strings.ToUpper(string(m.app.SourceLang))
	if src == "" || src == "AUTO" {
		src = "AUTO"
	}
	tgt := strings.ToUpper(string(m.app.TargetLang))
	if tgt == "" {
		tgt = "AUTO"
	}
	return src + " → " + tgt
}

func (m Model) renderHeader(cardWidth int) string {
	if cardWidth < 2 {
		return ""
	}
	logo := m.styles.Logo.Render("◆ tnow")

	quick := "Quick"
	analyze := "Analyze"
	var quickTab, analyzeTab string
	if m.mode == ModeQuick {
		quickTab = m.styles.ModeActive.Render(quick)
		analyzeTab = m.styles.ModeInactive.Render(analyze)
	} else {
		quickTab = m.styles.ModeInactive.Render(quick)
		analyzeTab = m.styles.ModeActive.Render(analyze)
	}
	modeCluster := lipgloss.JoinHorizontal(lipgloss.Center, quickTab, " ", analyzeTab)
	if cardWidth < 44 {
		if m.mode == ModeQuick {
			modeCluster = quickTab
		} else {
			modeCluster = analyzeTab
		}
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center, logo, "   ", modeCluster)

	modelLabel := strings.TrimSpace(m.app.Provider)
	if modelLabel == "" {
		modelLabel = strings.TrimSpace(m.app.Model)
	} else if m.app.Model != "" {
		modelLabel += " · " + m.app.Model
	}
	langPill := m.styles.BadgePill.Render(m.langPair())
	right := ""
	if cardWidth >= 24 {
		right = langPill
	}
	if cardWidth >= 72 && modelLabel != "" {
		modelLabel = ansi.Truncate(modelLabel, 28, "…")
		right = lipgloss.JoinHorizontal(lipgloss.Center, langPill, "  ", m.styles.HeaderMeta.Render(modelLabel))
	}

	if lipgloss.Width(left)+lipgloss.Width(right) > cardWidth {
		right = ""
	}
	if lipgloss.Width(left) > cardWidth {
		if cardWidth >= 30 {
			modeCluster = quickTab
			if m.mode == ModeAnalyze {
				modeCluster = analyzeTab
			}
			left = lipgloss.JoinHorizontal(lipgloss.Center, logo, "   ", modeCluster)
		} else {
			left = m.styles.Logo.Render("◆")
		}
	}
	if lipgloss.Width(left) > cardWidth {
		left = ansi.Truncate(left, cardWidth, "…")
	}

	spacing := max(1, cardWidth-lipgloss.Width(left)-lipgloss.Width(right))
	topRow := fitLine(left+strings.Repeat(" ", spacing)+right, cardWidth)
	divider := m.styles.SubtleLine.Render(strings.Repeat("─", cardWidth))
	return topRow + "\n" + divider
}

func (m Model) renderInputCard(cardWidth, cardHeight int) string {
	if cardHeight < 3 {
		cardHeight = 3
	}
	contentHeight := max(1, cardHeight-2)
	usableWidth := max(1, cardWidth-4)
	m.input.SetWidth(usableWidth)
	m.input.SetHeight(contentHeight)

	textareaLines := strings.Split(m.input.View(), "\n")
	if len(textareaLines) > contentHeight {
		textareaLines = textareaLines[:contentHeight]
	}
	for len(textareaLines) < contentHeight {
		textareaLines = append(textareaLines, "")
	}

	title := "Input"
	if m.mode == ModeAnalyze {
		title = "Word / Phrase"
	}
	focused := !m.focusOutput && !m.hasSuggestions()
	return m.titledCard(title, m.inputMeta(), textareaLines, cardWidth, cardHeight, focused)
}

func (m Model) inputMeta() string {
	value := m.input.Value()
	if strings.TrimSpace(value) == "" {
		return ""
	}
	lines := strings.Count(value, "\n") + 1
	if lines > 1 {
		return fmt.Sprintf("%d lines", lines)
	}
	chars := len([]rune(value))
	if chars == 1 {
		return "1 char"
	}
	return fmt.Sprintf("%d chars", chars)
}

// resultText returns the styled content for the output panel. Keeping this in
// one place lets the panel and the footer overflow hint agree on what is on
// screen.
func (m Model) resultText() string {
	switch {
	case m.mode == ModeAnalyze && m.analysisResult != nil:
		return RenderAnalysisStructured(*m.analysisResult, m.highlightQuery, m.styles, m.cardWidth())
	case m.result != "":
		return RenderHighlightedText(m.result, m.highlightQuery, m.styles)
	default:
		return ""
	}
}

func (m Model) resultOverflow() bool {
	_, resultHeight := m.layoutSizes()
	if resultHeight < 3 {
		return false
	}
	content := m.resultText()
	if strings.TrimSpace(content) == "" {
		return false
	}
	usableWidth := max(1, m.cardWidth()-4)
	return len(wrapANSI(content, usableWidth)) > resultHeight-2
}

func (m Model) resultPageStep() int {
	_, resultHeight := m.layoutSizes()
	return max(1, resultHeight-3)
}

func (m *Model) scrollResult(delta int) {
	if m.result == "" && !m.loading {
		return
	}
	m.focusOutput = true
	m.resultScroll = max(0, m.resultScroll+delta)
}

func (m Model) renderResultCard(cardWidth, cardHeight int) string {
	if cardHeight < 3 {
		cardHeight = 3
	}
	contentHeight := cardHeight - 2
	usableWidth := max(1, cardWidth-4)

	resLabel := "Translation"
	if m.mode == ModeAnalyze {
		resLabel = "Analysis"
	}

	content := m.resultText()
	if strings.TrimSpace(content) == "" {
		lines := make([]string, contentHeight)
		message := "Translation appears here"
		style := m.styles.EmptyState
		if m.loading {
			statusText := strings.ReplaceAll(m.status, "\n", " ")
			if statusText == "" {
				statusText = "Translating…"
			}
			message = m.spinner.View() + " " + statusText
			style = m.styles.StatusWarning
		} else if m.mode == ModeAnalyze {
			message = "Analysis appears here"
		}
		mid := contentHeight / 2
		if mid >= len(lines) {
			mid = len(lines) - 1
		}
		if mid >= 0 {
			lines[mid] = lipgloss.PlaceHorizontal(usableWidth, lipgloss.Center, style.Render(message))
		}
		return m.titledCard(resLabel, m.resultMeta(0, contentHeight, 0), lines, cardWidth, cardHeight, m.focusOutput)
	}

	wrapped := wrapANSI(content, usableWidth)
	maxScroll := max(0, len(wrapped)-contentHeight)
	scroll := m.resultScroll
	if m.loading {
		scroll = maxScroll
	}
	scroll = max(0, min(scroll, maxScroll))
	viewport := make([]string, contentHeight)
	for i := 0; i < contentHeight; i++ {
		idx := scroll + i
		if idx >= 0 && idx < len(wrapped) {
			viewport[i] = wrapped[idx]
		}
	}
	return m.titledCard(resLabel, m.resultMeta(scroll, contentHeight, len(wrapped)), viewport, cardWidth, cardHeight, m.focusOutput)
}

// resultMeta keeps the output border informative without changing its height.
func (m Model) resultMeta(scroll, visible, total int) string {
	if m.loading {
		if total > visible {
			last := min(total, scroll+visible)
			return fmt.Sprintf("%s %d–%d / %d", m.spinner.View(), scroll+1, last, total)
		}
		return ""
	}
	if total > visible {
		last := min(total, scroll+visible)
		return fmt.Sprintf("%d–%d / %d", scroll+1, last, total)
	}
	return m.langPair()
}

// renderCommandPopup builds the floating palette. Its dimensions are decided
// by the caller from terminal size only; the item count never resizes it.
func (m Model) renderCommandPopup(width, height int) string {
	kind, items, title, _ := m.suggestionItems()
	if kind == "" || title == "" {
		return ""
	}
	contentHeight := max(1, height-2)
	start, end := m.popupWindow(contentHeight, len(items))
	lines := m.popupLines(kind, max(1, width-4), contentHeight, start, end)
	return m.titledCard(title, m.popupMeta(start, end, len(items)), lines, width, height, true)
}

func (m Model) popupWindow(rows, total int) (start, end int) {
	if rows < 1 || total < 1 {
		return 0, 0
	}
	focus := m.suggestIdx
	if focus < 0 || focus >= total {
		focus = 0
	}
	if focus >= rows {
		start = focus - rows + 1
	}
	if start+rows > total {
		start = max(0, total-rows)
	}
	end = min(total, start+rows)
	return start, end
}

func (m Model) popupLines(kind string, width, rows, start, end int) []string {
	lines := make([]string, 0, rows)
	switch kind {
	case "command":
		suggestions := slashSuggestionsFor(m.input.Value())
		showDesc := width >= 48
		rowWidth := max(1, width-3)
		usageWidth := rowWidth
		if showDesc {
			usageWidth = min(30, max(10, rowWidth/2))
		}
		for i := start; i < end && i < len(suggestions); i++ {
			command := suggestions[i]
			usage := command.Usage
			description := command.Description
			if !showDesc {
				description = ""
			}
			if i == m.suggestIdx {
				plain := " ▸ " + fitLine(usage, usageWidth)
				if description != "" {
					plain += "  " + description
				}
				lines = append(lines, m.styles.PopupSelected.Render(fitLine(plain, width)))
				continue
			}
			line := "   " + fitLine(m.styles.PopupKey.Render(usage), usageWidth)
			if description != "" {
				line += "  " + m.styles.PopupDesc.Render(description)
			}
			lines = append(lines, fitLine(line, width))
		}
	case "language":
		_, query, _ := slashLanguageQuery(m.input.Value())
		suggestions := m.languageSuggestions(query)
		for i := start; i < end && i < len(suggestions); i++ {
			suggestion := suggestions[i]
			label := string(suggestion.Language) + " · " + suggestion.Name
			if suggestion.Region != "" && width >= 44 {
				label += " · " + suggestion.Region
			}
			if i == m.suggestIdx {
				lines = append(lines, m.styles.PopupSelected.Render(fitLine(" ▸ "+label, width)))
			} else {
				lines = append(lines, fitLine("   "+label, width))
			}
		}
	}
	for len(lines) < rows {
		lines = append(lines, "")
	}
	if len(lines) > rows {
		lines = lines[:rows]
	}
	return lines
}

func (m Model) popupMeta(start, end, total int) string {
	if total <= 0 {
		return ""
	}
	if total > end-start {
		return fmt.Sprintf("%d–%d / %d", start+1, end, total)
	}
	return fmt.Sprintf("%d", total)
}

// keyHint is one context-sensitive footer shortcut.
type keyHint struct {
	key  string
	desc string
}

func (m Model) footerHints() []keyHint {
	if m.hasSuggestions() {
		kind, _, _, _ := m.suggestionItems()
		pick := "Select"
		if kind == "language" {
			pick = "Apply"
		}
		return []keyHint{
			{key: "↑↓", desc: "Navigate"},
			{key: "Enter", desc: pick},
			{key: "Esc", desc: "Close"},
		}
	}
	if m.loading {
		return []keyHint{
			{key: "Esc", desc: "Cancel"},
			{key: "Ctrl+D", desc: "Quit"},
		}
	}
	if m.focusOutput && m.result != "" {
		hints := []keyHint{{key: "Ctrl+Y", desc: "Copy"}}
		if m.resultOverflow() {
			hints = append(hints, keyHint{key: "PgUp/PgDn", desc: "Scroll"})
		}
		return append(hints, keyHint{key: "Esc", desc: "Back"})
	}
	if m.highlightQuery != "" {
		hints := []keyHint{{key: "Ctrl+H", desc: "Edit"}}
		if m.result != "" {
			hints = append(hints, keyHint{key: "Ctrl+Y", desc: "Copy"})
		}
		return append(hints, keyHint{key: "Esc", desc: "Clear"})
	}
	if m.result != "" {
		hints := []keyHint{{key: "Ctrl+Y", desc: "Copy"}}
		if m.resultOverflow() {
			hints = append(hints, keyHint{key: "PgUp/PgDn", desc: "Scroll"})
		}
		if m.mode == ModeAnalyze {
			hints = append(hints, keyHint{key: "Alt+Enter", desc: "Retry"})
		} else {
			hints = append(hints, keyHint{key: "Alt+Enter", desc: "Retry"})
		}
		return append(hints, keyHint{key: "/", desc: "Commands"})
	}
	submit := "Translate"
	if m.mode == ModeAnalyze {
		submit = "Analyze"
	}
	return []keyHint{
		{key: "Alt+Enter", desc: submit},
		{key: "/", desc: "Commands"},
		{key: "Tab", desc: "Mode"},
		{key: "Ctrl+D", desc: "Quit"},
	}
}

func (m Model) renderHintLine(hints []keyHint, width int) string {
	render := func(items []keyHint, withDesc bool) string {
		parts := make([]string, 0, len(items))
		for _, hint := range items {
			part := m.styles.FooterKey.Render(hint.key)
			if withDesc && hint.desc != "" {
				part += " " + m.styles.FooterDesc.Render(hint.desc)
			}
			parts = append(parts, part)
		}
		return strings.Join(parts, "  ·  ")
	}
	if width <= 0 {
		return ""
	}
	full := render(hints, true)
	if lipgloss.Width(full) <= width {
		return full
	}
	keys := render(hints, false)
	if lipgloss.Width(keys) <= width {
		return keys
	}
	for n := len(hints); n > 0; n-- {
		candidate := render(hints[:n], false)
		if lipgloss.Width(candidate) <= width {
			return candidate
		}
	}
	return ""
}

// footerStatus returns the left-hand status label. The final bool marks a
// message that must not be dropped when the footer is narrow (errors, prompts,
// and in-flight work).
func (m Model) footerStatus() (string, lipgloss.Style, bool) {
	statusText := strings.ReplaceAll(m.status, "\n", " ")
	if m.loading {
		if statusText == "" {
			statusText = "Working…"
		}
		return m.spinner.View() + " " + statusText, m.styles.StatusWarning, true
	}
	if statusText != "" {
		noticePrefixes := []string{
			"target language set to ", "model set to ", "provider set to ", "word analysis mode",
			"quick translation mode", "filtering", "showing all", "auto-copied",
			"key saved for ",
		}
		for _, prefix := range noticePrefixes {
			if strings.HasPrefix(statusText, prefix) {
				return "✓ " + statusText, m.styles.StatusSuccess, true
			}
		}
		if strings.HasPrefix(statusText, "provider ") && strings.Contains(statusText, " · model ") {
			return "✓ " + statusText, m.styles.StatusSuccess, true
		}
		promptPrefixes := []string{"enter an API key", "add an API key", "choose a language"}
		for _, prefix := range promptPrefixes {
			if strings.HasPrefix(statusText, prefix) {
				return "→ " + statusText, m.styles.StatusWarning, true
			}
		}
		switch statusText {
		case "done":
			return "✓ Done", m.styles.StatusSuccess, true
		case "copied":
			return "✓ Copied to clipboard", m.styles.StatusSuccess, true
		case "cancelled":
			return "⊘ Request cancelled", m.styles.StatusWarning, true
		default:
			return "✗ " + statusText, m.styles.StatusDanger, true
		}
	}
	if m.highlightQuery != "" {
		return "Highlight: " + m.highlightQuery, m.styles.StatusNeutral, false
	}
	if m.hasSuggestions() {
		_, _, title, _ := m.suggestionItems()
		return title, m.styles.StatusNeutral, false
	}
	if m.focusOutput && m.result != "" {
		return "Result", m.styles.StatusNeutral, false
	}
	return "Ready", m.styles.StatusNeutral, false
}

// renderFooter is a single fixed row: contextual status on the left and only
// the shortcuts that apply to the current screen on the right.
func (m Model) renderFooter(cardWidth int) string {
	inner := max(1, cardWidth-2)
	status, style, important := m.footerStatus()
	hints := ""
	statusPart := ""

	if important {
		// Preserve actionable feedback first; shortcuts use whatever space is
		// left and disappear cleanly in a narrow split pane.
		statusPart = style.Render(ansi.Truncate(status, max(1, inner-2), "…"))
		budget := inner - lipgloss.Width(statusPart) - 2
		if budget >= 10 {
			hints = m.renderHintLine(m.footerHints(), budget)
		}
	} else {
		hints = m.renderHintLine(m.footerHints(), inner)
		available := inner - lipgloss.Width(hints) - 2
		if available >= 8 {
			statusPart = style.Render(ansi.Truncate(status, available, "…"))
		}
	}

	if statusPart == "" && hints == "" {
		return " " + strings.Repeat(" ", inner) + " "
	}
	if statusPart == "" {
		return " " + lipgloss.PlaceHorizontal(inner, lipgloss.Right, hints) + " "
	}
	gap := max(2, inner-lipgloss.Width(statusPart)-lipgloss.Width(hints))
	return " " + fitLine(statusPart+strings.Repeat(" ", gap)+hints, inner) + " "
}

func (m Model) renderModal(title, label, inputView, hint string) string {
	width := m.width
	if width == 0 {
		width = 80
	}
	boxWidth := min(64, width-4)
	if boxWidth < 30 || boxWidth > width-2 {
		boxWidth = max(10, width-2)
	}
	innerWidth := max(1, boxWidth-4)

	labelLines := wrapANSI(m.styles.BoxLabel.Render(label), innerWidth)
	hintLines := wrapANSI(m.styles.ModalHint.Render(hint), innerWidth)
	inputLines := strings.Split(inputView, "\n")

	lines := []string{""}
	lines = append(lines, labelLines...)
	lines = append(lines, "")
	lines = append(lines, inputLines...)
	lines = append(lines, "")
	lines = append(lines, hintLines...)

	modalBox := m.titledCard(title, "", lines, boxWidth, len(lines)+2, true)
	return m.centeredOverlay(modalBox, ColorBgRaised)
}

// renderAPIKeyModal is intentionally compact: the provider, the masked key,
// the destination file, and one-line hints. It never shifts the underlying
// layout because it is rendered as a centered overlay.
func (m Model) renderAPIKeyModal() string {
	width := m.width
	if width == 0 {
		width = 80
	}
	boxWidth := min(64, width-4)
	if boxWidth < 36 || boxWidth > width-2 {
		boxWidth = max(10, width-2)
	}
	innerWidth := max(1, boxWidth-4)
	m.apiKeyInput.Width = max(8, innerWidth-6)

	provider := m.pendingProvider
	if provider == "" {
		provider = m.app.Provider
	}
	status := "API key required"
	if m.app.HasAPIKey(provider) {
		status = "key configured · paste a new one to replace it"
	} else if !config.ProviderRequiresKey(provider) {
		status = "API key optional for this provider"
	}

	lines := []string{
		"",
		m.styles.HighlightBold.Render(provider),
		fitLine(m.styles.BoxMeta.Render(status), innerWidth),
		"",
	}
	inputView := strings.TrimRight(m.apiKeyInput.View(), "\n")
	lines = append(lines, strings.Split(inputView, "\n")...)
	if m.apiKeyErr != "" {
		lines = append(lines, "")
		errorText := strings.ReplaceAll(m.apiKeyErr, "\n", " ")
		lines = append(lines, fitLine(m.styles.StatusDanger.Render("✗ "+errorText), innerWidth))
	}
	lines = append(lines, "")
	lines = append(lines, fitLine(m.styles.BoxMeta.Render("saves to "+prettyConfigPath()), innerWidth))
	lines = append(lines, fitLine(m.styles.ModalHint.Render("Enter save  ·  Esc cancel  ·  input is hidden"), innerWidth))

	modalBox := m.titledCard("Connect Provider", "masked input", lines, boxWidth, len(lines)+2, true)
	return m.centeredOverlay(modalBox, ColorBgRaised)
}

func formatAnalysis(r analysis.Result) string {
	var b strings.Builder
	b.WriteString(r.Word)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 28))
	b.WriteString("\n\n")
	if r.IPA != "" {
		b.WriteString(r.IPA)
		b.WriteString("\n\n")
	}
	if r.PartOfSpeech != "" {
		b.WriteString(strings.ToUpper(r.PartOfSpeech))
		b.WriteString("\n\n")
	}
	b.WriteString("Meanings\n\n")
	for _, d := range r.Definitions {
		b.WriteString("• " + d.Meaning + "\n")
		if len(d.Translations) > 0 {
			b.WriteString("  " + strings.Join(d.Translations, ", ") + "\n")
		}
	}
	if len(r.Examples) > 0 {
		b.WriteString("\nExamples\n\n")
		for _, e := range r.Examples {
			b.WriteString(e.Source + "\n→ " + e.Translation + "\n\n")
		}
	}
	return strings.TrimSpace(b.String())
}
