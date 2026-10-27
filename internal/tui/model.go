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

	// result viewport scroll
	resultScroll int

	styles Styles
}

func NewModel(a *app.App) Model {
	styles := DefaultStyles()

	ta := textarea.New()
	ta.Placeholder = "Type or paste text…  Alt+Enter to translate"
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.SetHeight(4)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted))
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextPrimary))
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextDim))
	ta.BlurredStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextSecondary))

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
	if w > 92 {
		w = 92
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
// Nothing about the draft, the slash menu, the model list, or the result can
// change these values, which is what keeps the screen from jumping while the
// user types.
//
// The main screen is assembled as:
//
//	header (2) + blank (1) + input (H) + [blank (1) + result (R)] + status (1) + footer (1)
func (m Model) layoutSizes() (inputHeight, resultHeight int) {
	height := m.effectiveHeight()
	if height < 10 {
		return max(3, height-5), 0
	}

	body := height - 6
	if body < 8 {
		// Not enough room for a useful result card: let the editor fill the
		// body. The footer is only omitted in extremely short panes.
		return max(3, height-5), 0
	}

	target := body * 2 / 5
	target = max(7, min(target, 12))
	if body-target < 5 {
		return max(3, height-5), 0
	}
	return target, body - target
}

// inputContentHeight is the number of text lines inside the input frame.
func (m Model) inputContentHeight() int {
	inputHeight, _ := m.layoutSizes()
	return max(1, inputHeight-2)
}

// inlineMenuHeight is the number of rows reserved for the command menu inside
// the input card when there is no result card to use. It is fixed while the
// menu is open so filtering cannot resize the editor.
func (m Model) inlineMenuHeight() int {
	_, resultHeight := m.layoutSizes()
	if resultHeight > 0 || !m.hasSuggestions() {
		return 0
	}
	inputHeight, _ := m.layoutSizes()
	return min(3, max(1, (inputHeight-2)/2))
}

// textareaContentHeight is the editor viewport height inside the fixed input
// card. In very short panes the command menu takes a few rows, but the card
// itself never changes size.
func (m Model) textareaContentHeight() int {
	content := m.inputContentHeight()
	if menu := m.inlineMenuHeight(); menu > 0 && menu < content {
		content -= menu
	}
	return max(1, content)
}

// syncInputSize keeps the real textarea viewport in sync with the fixed card.
// It is called from Update so cursor scrolling uses the same dimensions that
// View renders with, instead of changing the textarea only on a copy.
func (m Model) syncInputSize() Model {
	cardWidth := m.cardWidth()
	usableWidth := max(1, cardWidth-4)
	m.input.SetWidth(usableWidth)
	previousHeight := m.input.Height()
	m.input.SetHeight(m.textareaContentHeight())
	if m.input.Height() != previousHeight {
		// SetHeight changes the viewport without scrolling it. A no-op key
		// update lets the textarea re-anchor the viewport around the cursor,
		// so opening the inline command menu never hides the line being typed.
		m.input, _ = m.input.Update(tea.KeyMsg{Type: tea.KeyRunes})
	}
	return m
}

// suggestionItems returns the current suggestion list and its kind.
func (m Model) suggestionItems() (kind string, items []string, title, hint string) {
	if _, query, ok := slashLanguageQuery(m.input.Value()); ok {
		suggestions := language.Suggest(query)
		if len(suggestions) > 0 {
			for _, suggestion := range suggestions {
				items = append(items, string(suggestion.Language))
			}
			return "language", items, "LANGUAGES", "↑↓ select · Tab accept · Enter apply"
		}
	}

	sugs := slashSuggestionsFor(m.input.Value())
	if len(sugs) == 0 {
		return "", nil, "", ""
	}
	title = "COMMANDS"
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
			m.status = ""
			m.input.Focus()
			return m, nil

		case "esc":
			if m.highlightQuery != "" {
				m.highlightQuery = ""
				m.highlightInput.SetValue("")
				return m, nil
			}
			if m.suggestIdx != -1 {
				m.suggestIdx = -1
				return m, nil
			}
			if m.loading && m.cancel != nil {
				m.cancel()
				m.loading = false
				m.status = "cancelled"
				return m, nil
			}
			return m, nil

		case "tab":
			if m.mode == ModeQuick {
				m.mode = ModeAnalyze
				m.input.Placeholder = "Type a word or short phrase…  Alt+Enter to analyze"
			} else {
				m.mode = ModeQuick
				m.input.Placeholder = "Type or paste text…  Alt+Enter to translate"
			}
			m.result = ""
			m.analysisResult = nil
			m.resultScroll = 0
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

		case "pgup", "ctrl+u", "alt+up":
			if m.resultScroll > 0 {
				m.resultScroll--
				if m.resultScroll < 0 {
					m.resultScroll = 0
				}
			}
			return m, nil
		case "pgdown", "alt+down", "ctrl+down":
			m.resultScroll++
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
		m.status = msg.err.Error()
		m.cancel = nil
		return m, nil

	case streamDoneMsg:
		if !m.loading {
			return m, nil
		}
		m.loading = false
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
			m.status = msg.err.Error()
			return m, nil
		}
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
	// The draft may have just added/removed a slash command, which changes
	// the inline menu reservation in very short panes.
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
			m.input.Placeholder = "Type a word or short phrase…  Alt+Enter to analyze"
			m.status = "word analysis mode"
		} else {
			if m.mode == ModeQuick {
				m.mode = ModeAnalyze
				m.input.Placeholder = "Type a word or short phrase…  Alt+Enter to analyze"
				m.status = "word analysis mode"
			} else {
				m.mode = ModeQuick
				m.input.Placeholder = "Type or paste text…  Alt+Enter to translate"
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
	height := m.height
	if height == 0 {
		height = 24
	}
	cardWidth := m.cardWidth()

	// Overlays take priority and stay in the same centered frame language as
	// the main screen.
	if m.showAPIKey {
		return m.renderAPIKeyModal()
	}
	if m.showModel {
		m.modelInput.Width = max(12, min(54, m.width-24))
		return m.renderModal(
			"Configure Model",
			"Enter model identifier:",
			m.modelInput.View(),
			"Enter to save  •  Esc to cancel",
		)
	}
	if m.showHighlight {
		m.highlightInput.Width = max(12, min(54, m.width-24))
		return m.renderModal(
			"Highlight Sections / Terms",
			"Enter section name, word, or pattern to highlight:",
			m.highlightInput.View(),
			"Enter to apply  •  Esc to dismiss",
		)
	}
	if m.showModelsBrowser {
		return m.renderModelsBrowser()
	}

	// Input and result heights are fixed for the current terminal size. The
	// slash menu never adds rows to the input card: in normal layouts it uses
	// the result slot, and in very short panes it replaces a few editor rows
	// inside the same fixed frame.
	inputHeight, resultHeight := m.layoutSizes()
	suggestionsVisible := m.hasSuggestions()
	var inlineMenu []string
	if suggestionsVisible && resultHeight == 0 {
		inlineMenu = m.inlineCommandLines(cardWidth-4, m.inlineMenuHeight())
	}

	headerBar := m.renderHeader(cardWidth)
	inputFrame := m.renderInputCard(cardWidth, inputHeight, inlineMenu)

	statusLine := m.renderStatusLine()
	if statusLine == "" {
		statusLine = " "
	}
	footer := m.renderFooter(cardWidth)

	var b strings.Builder
	b.WriteString(headerBar)
	b.WriteString("\n\n")
	b.WriteString(inputFrame)
	if resultHeight > 0 {
		b.WriteString("\n\n")
		if suggestionsVisible {
			b.WriteString(m.renderCommandCard(cardWidth, resultHeight))
		} else {
			b.WriteString(m.renderResultCard(cardWidth, resultHeight))
		}
	}
	b.WriteString("\n")
	b.WriteString(fitLine(statusLine, cardWidth))
	if height >= 8 {
		b.WriteString("\n")
		b.WriteString(fitLine(footer, cardWidth))
	}

	content := b.String()
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	content = strings.Join(lines, "\n")
	if width > 0 {
		content = lipgloss.PlaceHorizontal(width, lipgloss.Center, content)
	}
	if height > 0 {
		content = lipgloss.PlaceVertical(height, lipgloss.Center, content)
	}
	return content
}

// pagedSuggestionLines renders an arbitrary suggestion list into a fixed
// number of rows. The window follows the selected row and reports overflow in
// the hint row without changing the surrounding card height.
func (m Model) pagedSuggestionLines(width, maxLines, total int, hint string, row func(int) string) []string {
	if maxLines <= 0 || total <= 0 {
		return nil
	}

	hintRows := 0
	if maxLines >= 4 {
		hintRows = 1
	}
	listRows := maxLines - hintRows
	if listRows < 1 {
		listRows = maxLines
		hintRows = 0
	}

	focus := m.suggestIdx
	if focus < 0 || focus >= total {
		focus = 0
	}
	start := 0
	if focus >= listRows {
		start = focus - listRows + 1
	}
	if start+listRows > total {
		start = max(0, total-listRows)
	}
	end := min(total, start+listRows)

	lines := make([]string, 0, maxLines)
	for i := start; i < end; i++ {
		lines = append(lines, row(i))
	}
	if hintRows > 0 && len(lines) < maxLines {
		if width < 56 {
			hint = "↑↓ · Tab · Esc"
		}
		if width < 38 {
			hint = "↑↓ · Tab"
		}
		if start > 0 || end < total {
			hint += fmt.Sprintf("  %d-%d/%d", start+1, end, total)
		}
		lines = append(lines, fitLine(m.styles.BoxMeta.Render("  "+hint), width))
	}
	return lines
}

// suggestionLines dispatches to the command or language renderer.
func (m Model) suggestionLines(width, maxLines int) []string {
	kind, _, _, _ := m.suggestionItems()
	if kind == "language" {
		return m.languageLines(width, maxLines)
	}
	return m.commandLines(width, maxLines)
}

// commandLines renders the slash command list.
func (m Model) commandLines(width, maxLines int) []string {
	suggestions := slashSuggestionsFor(m.input.Value())
	if len(suggestions) == 0 {
		return nil
	}
	return m.pagedSuggestionLines(width, maxLines, len(suggestions), "↑↓ select · Tab accept · Esc close", func(i int) string {
		command := suggestions[i]
		usage := command.Usage
		description := command.Description
		if width < 54 {
			description = ""
		}
		if i == m.suggestIdx {
			line := " ▸ " + usage
			if description != "" {
				line += "  " + description
			}
			return m.styles.SuggestActive.Render(fitLine(line, width))
		}
		line := "  " + m.styles.SuggestIdle.Render(usage)
		if description != "" {
			line += "  " + m.styles.BoxMeta.Render(description)
		}
		return fitLine(line, width)
	})
}

// languageLines renders /to suggestions. Each row shows the code, display
// name, and a representative region so country-name input is easy to verify.
func (m Model) languageLines(width, maxLines int) []string {
	_, query, ok := slashLanguageQuery(m.input.Value())
	if !ok {
		return nil
	}
	suggestions := language.Suggest(query)
	if len(suggestions) == 0 {
		return nil
	}
	return m.pagedSuggestionLines(width, maxLines, len(suggestions), "↑↓ select · Tab accept · Enter apply", func(i int) string {
		suggestion := suggestions[i]
		label := string(suggestion.Language) + " · " + suggestion.Name
		if suggestion.Region != "" && width >= 44 {
			label += " · " + suggestion.Region
		}
		if i == m.suggestIdx {
			return m.styles.SuggestActive.Render(fitLine(" ▸ "+label, width))
		}
		return fitLine("  "+label, width)
	})
}

// inlineCommandLines is the short-pane fallback. It keeps the suggestion menu
// inside the fixed input frame when there is no result card to borrow.
func (m Model) inlineCommandLines(width, maxLines int) []string {
	if maxLines <= 0 {
		return nil
	}
	if maxLines == 1 {
		return m.suggestionLines(width, 1)
	}
	_, _, title, _ := m.suggestionItems()
	if title == "" {
		title = "COMMANDS"
	}
	lines := []string{fitLine(m.styles.BoxLabel.Render(title), width)}
	lines = append(lines, m.suggestionLines(width, maxLines-1)...)
	for len(lines) < maxLines {
		lines = append(lines, "")
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return lines
}

// renderCommandCard replaces the result card while a suggestion menu is open.
// It has exactly the same dimensions as the result card, so opening or closing
// the menu cannot move any other part of the screen.
func (m Model) renderCommandCard(cardWidth, cardHeight int) string {
	if cardHeight < 3 {
		cardHeight = 3
	}
	contentHeight := max(1, cardHeight-2)
	lines := m.suggestionLines(cardWidth-4, contentHeight)

	kind, items, title, _ := m.suggestionItems()
	if kind == "" || title == "" {
		title = "COMMANDS"
	}
	// When every match does not fit, put the total in the title so the
	// truncation is explicit without spending a list row on it.
	rows := contentHeight
	if rows >= 4 {
		rows--
	}
	if len(items) > rows {
		title = fmt.Sprintf("%s · %d", title, len(items))
	}
	return m.titledCard(title, lines, cardWidth, cardHeight, true)
}

func (m Model) renderHeader(cardWidth int) string {
	logo := m.styles.Logo.Render("◆ tnow")

	var quickMode, analyzeMode string
	if m.mode == ModeQuick {
		quickMode = m.styles.ModeActive.Render("Quick")
		analyzeMode = m.styles.ModeInactive.Render("Analyze")
	} else {
		quickMode = m.styles.ModeInactive.Render("Quick")
		analyzeMode = m.styles.ModeActive.Render("Analyze")
	}
	modeCluster := lipgloss.JoinHorizontal(lipgloss.Center, quickMode, " ", analyzeMode)
	if cardWidth < 50 {
		// In a narrow split pane keep the language badge visible instead of
		// spending the row on the inactive mode.
		if m.mode == ModeQuick {
			modeCluster = quickMode
		} else {
			modeCluster = analyzeMode
		}
	}

	src := strings.ToUpper(string(m.app.SourceLang))
	if src == "" || src == "AUTO" {
		src = "AUTO"
	}
	tgt := strings.ToUpper(string(m.app.TargetLang))
	langPill := m.styles.BadgePill.Render(fmt.Sprintf("%s → %s", src, tgt))

	left := lipgloss.JoinHorizontal(lipgloss.Center, logo, "   ", modeCluster)
	right := langPill

	modelLabel := strings.TrimSpace(m.app.Provider)
	if modelLabel == "" {
		modelLabel = strings.TrimSpace(m.app.Model)
	} else if m.app.Model != "" {
		modelLabel += " · " + m.app.Model
	}
	if cardWidth >= 68 && modelLabel != "" {
		runes := []rune(modelLabel)
		if len(runes) > 26 {
			modelLabel = string(runes[:25]) + "…"
		}
		modelPill := m.styles.BadgePill.Render(modelLabel)
		right = lipgloss.JoinHorizontal(lipgloss.Center, langPill, "  ", modelPill)
	}

	spacing := cardWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if spacing < 2 {
		right = ""
		spacing = cardWidth - lipgloss.Width(left)
	}
	if spacing < 0 {
		spacing = 0
	}
	topRow := fitLine(left+strings.Repeat(" ", spacing)+right, cardWidth)
	divider := m.styles.SubtleLine.Render(strings.Repeat("─", cardWidth))
	return topRow + "\n" + divider
}

func (m Model) renderInputCard(cardWidth, cardHeight int, menuLines []string) string {
	if cardHeight < 3 {
		cardHeight = 3
	}
	contentHeight := max(1, cardHeight-2)

	if len(menuLines) >= contentHeight {
		menuLines = menuLines[:max(0, contentHeight-1)]
	}
	textareaHeight := max(1, contentHeight-len(menuLines))

	usableWidth := max(1, cardWidth-4)
	m.input.SetWidth(usableWidth)
	m.input.SetHeight(textareaHeight)

	textareaLines := strings.Split(m.input.View(), "\n")
	if len(textareaLines) > textareaHeight {
		textareaLines = textareaLines[:textareaHeight]
	}
	for len(textareaLines) < textareaHeight {
		textareaLines = append(textareaLines, "")
	}

	title := "INPUT"
	if m.mode == ModeAnalyze {
		title = "WORD / PHRASE"
	}
	lines := make([]string, 0, contentHeight)
	lines = append(lines, textareaLines...)
	lines = append(lines, menuLines...)
	return m.titledCard(title, lines, cardWidth, cardHeight, true)
}

func (m Model) renderResultCard(cardWidth, cardHeight int) string {
	if cardHeight < 3 {
		cardHeight = 3
	}
	contentHeight := cardHeight - 2
	usableWidth := cardWidth - 4
	if usableWidth < 1 {
		usableWidth = 1
	}

	resLabel := "TRANSLATION"
	if m.mode == ModeAnalyze {
		resLabel = "ANALYSIS"
	}

	var content string
	switch {
	case m.loading:
		statusText := m.status
		if statusText == "" {
			statusText = "processing..."
		}
		loader := m.spinner.View() + " " + m.styles.StatusWarning.Render(statusText)
		if m.result != "" {
			content = loader + "\n\n" + RenderHighlightedText(m.result, m.highlightQuery, m.styles)
		} else {
			content = loader
		}
	case m.mode == ModeAnalyze && m.analysisResult != nil:
		content = RenderAnalysisStructured(*m.analysisResult, m.highlightQuery, m.styles, cardWidth)
	case m.result != "":
		content = RenderHighlightedText(m.result, m.highlightQuery, m.styles)
	}

	if strings.TrimSpace(content) == "" {
		lines := make([]string, contentHeight)
		message := "Translation will appear here · Alt+Enter to submit"
		if m.mode == ModeAnalyze {
			message = "Analysis will appear here · Alt+Enter to submit"
		}
		mid := contentHeight / 2
		if mid >= len(lines) {
			mid = len(lines) - 1
		}
		if mid >= 0 {
			lines[mid] = lipgloss.PlaceHorizontal(usableWidth, lipgloss.Center, m.styles.EmptyState.Render(message))
		}
		return m.titledCard(resLabel, lines, cardWidth, cardHeight, false)
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
	return m.titledCard(resLabel, viewport, cardWidth, cardHeight, false)
}

func (m Model) renderStatusLine() string {
	statusText := strings.ReplaceAll(m.status, "\n", " ")
	if m.loading {
		if statusText == "" {
			statusText = "processing..."
		}
		return m.spinner.View() + " " + m.styles.StatusWarning.Render(statusText)
	}
	if statusText != "" {
		noticePrefixes := []string{
			"target language set to ", "model set to ", "provider set to ", "word analysis mode",
			"quick translation mode", "filtering", "showing all", "provider ", "auto-copied",
			"key saved for ",
		}
		for _, prefix := range noticePrefixes {
			if strings.HasPrefix(statusText, prefix) {
				return m.styles.StatusSuccess.Render("✓ " + statusText)
			}
		}
		promptPrefixes := []string{"enter an API key", "add an API key", "choose a language"}
		for _, prefix := range promptPrefixes {
			if strings.HasPrefix(statusText, prefix) {
				return m.styles.StatusWarning.Render("→ " + statusText)
			}
		}
		switch statusText {
		case "done":
			return m.styles.StatusSuccess.Render("✓ Done")
		case "copied":
			return m.styles.StatusSuccess.Render("✓ Copied to clipboard")
		case "cancelled":
			return m.styles.StatusWarning.Render("⊘ Request cancelled")
		default:
			return m.styles.StatusDanger.Render("✗ " + statusText)
		}
	}
	if m.highlightQuery != "" {
		return m.styles.BoxMeta.Render("highlight: " + m.highlightQuery + " · Esc to clear")
	}
	value := strings.TrimSpace(m.input.Value())
	if value != "" {
		lines := strings.Count(m.input.Value(), "\n") + 1
		if lines > m.inputContentHeight() {
			return m.styles.BoxMeta.Render(fmt.Sprintf("↕ %d lines · editor scrolls", lines))
		}
	}
	return ""
}

// renderFooter is always present so the bottom of the screen never moves.
// The text the user sees at wide sizes is the canonical keybinding line.
func (m Model) renderFooter(cardWidth int) string {
	footerText := "Alt+Enter translate · / commands · Tab mode · Ctrl+E key · Ctrl+Y copy · Ctrl+D quit"
	if cardWidth < 84 {
		footerText = "Alt+Enter translate · / commands · Ctrl+E key · Ctrl+D quit"
	}
	if cardWidth < 64 {
		footerText = "Alt+Enter · / commands · Ctrl+E key"
	}
	if cardWidth < 40 {
		footerText = "Alt+Enter · /"
	}
	if m.highlightQuery != "" {
		footerText = "highlight: " + m.highlightQuery + " · Esc clear"
	}
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted)).Render(footerText)
	return fitLine(lipgloss.PlaceHorizontal(cardWidth, lipgloss.Center, styled), cardWidth)
}

func (m Model) renderModal(title, label, inputView, hint string) string {
	width := m.width
	if width == 0 {
		width = 80
	}
	height := m.height
	if height == 0 {
		height = 24
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

	modalBox := m.titledCard(title, lines, boxWidth, len(lines)+2, true)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modalBox)
}

// renderAPIKeyModal is intentionally compact: the provider, the masked key,
// the destination file, and one-line hints. It never shifts the underlying
// layout because it is rendered as a centered overlay.
func (m Model) renderAPIKeyModal() string {
	width := m.width
	if width == 0 {
		width = 80
	}
	height := m.height
	if height == 0 {
		height = 24
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

	modalBox := m.titledCard("Connect Provider", lines, boxWidth, len(lines)+2, true)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modalBox)
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
