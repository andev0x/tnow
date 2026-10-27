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

type managerKind int

const (
	managerNone managerKind = iota
	managerProviders
	managerModels
)

const maxSuggestionRows = 5

// fixedInputFrameInnerLines is the inner content height of the input frame.
// It reserves space for header(1) + textarea(4) + ghost(1) + divider(1) + suggestions(7) = 14
// so the slash menu appears inside the frame without shifting the result card.
const fixedInputFrameInnerLines = 14

// slashCommand describes a locally handled command shown in the suggestion
// menu while the user is typing a "/" command.
type slashCommand struct {
	Name        string
	Aliases     []string
	Usage       string
	Description string
}

var slashCommands = []slashCommand{
	{Name: "/to", Aliases: []string{"/lang", "/language", "/target"}, Usage: "/to <lang>", Description: "change target language"},
	{Name: "/model", Usage: "/model <name>", Description: "switch active model"},
	{Name: "/provider", Usage: "/provider <name>", Description: "switch active provider"},
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

// slashSuggestionsFor returns the commands matching the current slash input.
// It works on the effective slash line so that "hello\n/mo" still offers
// "/model" completion, and suggestions disappear once arguments start being typed.
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

// slashArgQuery reports whether value is a provider/model command and returns
// the optional argument used to filter the list. It inspects the effective
// slash line to support multiline input like "text to translate\n/model gpt".
func slashArgQuery(value string) (cmd, arg string, ok bool) {
	line := effectiveSlashLine(value)
	if !strings.HasPrefix(line, "/") {
		return "", "", false
	}
	if i := strings.IndexAny(line, " \t"); i >= 0 {
		cmd = strings.ToLower(line[:i])
		arg = strings.TrimSpace(line[i+1:])
	} else {
		cmd = strings.ToLower(line)
	}
	switch cmd {
	case "/provider", "/model":
		return cmd, arg, true
	}
	return "", "", false
}

func filterByPrefix(items []string, prefix string) []string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		return items
	}
	var filtered []string
	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item), prefix) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

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
	apiKeyInput     textinput.Model
	showAPIKey      bool
	pendingProvider string
	apiKeyErr       string

	// languages
	langs   []language.Language
	langIdx int

	// provider/model manager
	manager    managerKind
	modelCache map[string][]string

	// autocomplete
	suggestIdx      int
	filterExcellent bool

	styles Styles
}

func NewModel(a *app.App) Model {
	styles := DefaultStyles()

	ta := textarea.New()
	ta.Placeholder = "Type or paste text to translate... (Alt+Enter to submit)"
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
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPrimary)).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextPrimary))
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted))

	hi := textinput.New()
	hi.Placeholder = "type word, phrase, or section to highlight..."
	hi.CharLimit = 64
	hi.Prompt = "Highlight: "
	hi.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorHighlightFg)).Bold(true)
	hi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorHighlightFg)).Bold(true)
	hi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted))

	ai := textinput.New()
	ai.Placeholder = "paste API key (input hidden)"
	ai.CharLimit = 256
	ai.EchoMode = textinput.EchoNone
	ai.EchoCharacter = '•'
	ai.Prompt = "Key: "
	ai.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning)).Bold(true)
	ai.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextPrimary))
	ai.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted))

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
		spinner:         sp,
		langs:           langs,
		langIdx:         idx,
		modelCache:      make(map[string][]string),
		styles:          styles,
		suggestIdx:      -1,
		filterExcellent: false,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m Model) cardWidth() int {
	w := m.width - 6
	if w > 86 {
		w = 86
	}
	if w < 54 {
		w = 54
	}
	return w
}

// suggestionItems returns the current suggestion list and its kind.
func (m *Model) suggestionItems() (kind string, items []string, title, hint string) {
	val := m.input.Value()
	if cmd, arg, ok := slashArgQuery(val); ok {
		switch cmd {
		case "/provider":
			title = "PROVIDERS"
			items = filterByPrefix(m.providerNames(), arg)
			hint = "↑↓ navigate · Tab/Enter select · Esc dismiss"
			return "provider", items, title, hint
		case "/model":
			title = "MODELS · " + strings.ToUpper(m.app.Provider)
			var all []string
			if m.filterExcellent {
				all = m.app.ModelsForProviderFiltered(m.app.Provider, true)
				// merge cached models filtered
				cached := filterByPrefix(m.modelsForProvider(m.app.Provider), "")
				var excellentCached []string
				for _, c := range cached {
					if app.IsTranslationExcellent(c) {
						excellentCached = append(excellentCached, c)
					}
				}
				// dedup merge
				seen := map[string]bool{}
				for _, v := range all {
					seen[v] = true
				}
				for _, v := range excellentCached {
					if !seen[v] {
						all = append(all, v)
					}
				}
				sort.Strings(all)
				// re-sort excellent first
				var ex, other []string
				for _, v := range all {
					if app.IsTranslationExcellent(v) {
						ex = append(ex, v)
					} else {
						other = append(other, v)
					}
				}
				sort.Strings(ex)
				sort.Strings(other)
				all = append(ex, other...)
			} else {
				all = m.modelsForProvider(m.app.Provider)
				// prioritize translation excellent
				var ex, other []string
				for _, v := range all {
					if app.IsTranslationExcellent(v) {
						ex = append(ex, v)
					} else {
						other = append(other, v)
					}
				}
				sort.Strings(ex)
				sort.Strings(other)
				all = append(ex, other...)
			}
			items = filterByPrefix(all, arg)
			hint = "↑↓ navigate · Tab/Enter select"
			if m.filterExcellent {
				hint += " · Ctrl+T show all"
			} else {
				hint += " · Ctrl+T filter translation"
			}
			return "model", items, title, hint
		}
	}
	sugs := slashSuggestionsFor(val)
	if len(sugs) > 0 {
		title = "COMMANDS"
		for _, s := range sugs {
			items = append(items, s.Name)
		}
		hint = "↑↓ navigate · Tab/Enter autocomplete · Esc dismiss"
		return "command", items, title, hint
	}
	return "", nil, "", ""
}

func (m *Model) hasSuggestions() bool {
	kind, items, _, _ := m.suggestionItems()
	if kind == "" {
		return false
	}
	// also consider manager overlay when no slash prefix
	if m.manager != managerNone && kind == "" {
		return true
	}
	return len(items) > 0 || kind == "command"
}

func (m *Model) ghostText() string {
	kind, items, _, _ := m.suggestionItems()
	if len(items) == 0 {
		return ""
	}
	val := effectiveSlashLine(m.input.Value())
	if kind == "command" {
		// for "/mo" ghost "del"
		fields := strings.Fields(val)
		if len(fields) != 1 {
			return ""
		}
		prefix := strings.ToLower(fields[0])
		for _, cmd := range slashCommands {
			if cmd.matches(prefix) && cmd.Name != prefix {
				if strings.HasPrefix(strings.ToLower(cmd.Name), prefix) {
					return cmd.Name[len(prefix):]
				}
				for _, alias := range cmd.Aliases {
					if strings.HasPrefix(strings.ToLower(alias), prefix) {
						return alias[len(prefix):]
					}
				}
			}
		}
		// fallback: first item suffix
		if strings.HasPrefix(strings.ToLower(items[0]), prefix) {
			return items[0][len(prefix):]
		}
		return ""
	}
	if kind == "provider" || kind == "model" {
		// value is "/provider g" -> ghost is remainder of first match
		parts := strings.SplitN(val, " ", 2)
		var arg string
		if len(parts) == 2 {
			arg = strings.TrimSpace(parts[1])
		}
		if arg == "" {
			return ""
		}
		lowerArg := strings.ToLower(arg)
		for _, it := range items {
			if strings.HasPrefix(strings.ToLower(it), lowerArg) && strings.ToLower(it) != lowerArg {
				return it[len(arg):]
			}
		}
	}
	return ""
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		cardWidth := m.cardWidth()
		// inner width for textarea inside rounded border with padding
		innerWidth := cardWidth - 4
		if innerWidth < 20 {
			innerWidth = 20
		}
		m.input.SetWidth(innerWidth)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// API key overlay has highest priority
		if m.showAPIKey {
			switch msg.String() {
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
				m.showAPIKey = false
				m.apiKeyInput.SetValue("")
				m.apiKeyErr = ""
				m.pendingProvider = ""
				m.app.SetProvider(prov)
				m.status = "provider " + prov + " connected"
				m.input.SetValue("")
				return m, m.fetchModelsCmd(prov)
			case "esc":
				m.showAPIKey = false
				m.apiKeyInput.SetValue("")
				m.apiKeyErr = ""
				m.pendingProvider = ""
				m.status = "cancelled"
				return m, nil
			}
			var cmd tea.Cmd
			m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
			return m, cmd
		}

		// Model overlay
		if m.showModel {
			switch msg.String() {
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

		// Suggestion navigation when suggestions visible
		if m.hasSuggestions() {
			kind, items, _, _ := m.suggestionItems()
			// Ctrl+T toggles translation filter for model suggestions
			if msg.String() == "ctrl+t" && kind == "model" {
				m.filterExcellent = !m.filterExcellent
				m.suggestIdx = -1
				if m.filterExcellent {
					m.status = "filtering translation-optimized models"
				} else {
					m.status = "showing all models"
				}
				return m, nil
			}
			switch msg.String() {
			case "up", "ctrl+p":
				if len(items) > 0 {
					if m.suggestIdx == -1 {
						m.suggestIdx = len(items) - 1
					} else {
						m.suggestIdx--
						m.clampSuggestIdx(len(items))
					}
					return m, nil
				}
			case "down", "ctrl+n":
				if len(items) > 0 {
					if m.suggestIdx == -1 {
						m.suggestIdx = 0
					} else {
						m.suggestIdx++
						m.clampSuggestIdx(len(items))
					}
					return m, nil
				}
			case "tab":
				if len(items) > 0 {
					idx := m.suggestIdx
					if idx < 0 || idx >= len(items) {
						idx = 0
					}
					m.applySuggestion(items[idx])
					m.suggestIdx = -1
					return m, nil
				}
			case "esc":
				if m.suggestIdx != -1 {
					m.suggestIdx = -1
					return m, nil
				}
				if m.manager != managerNone {
					m.manager = managerNone
					return m, nil
				}
				// dismiss suggestions by clearing slash? just reset idx
				m.suggestIdx = -1
				// let Esc also clear highlight etc. fall through not
			}
			// Enter with suggestion selected should autocomplete, not submit
			if msg.String() == "enter" {
				if m.suggestIdx >= 0 && m.suggestIdx < len(items) {
					m.applySuggestion(items[m.suggestIdx])
					m.suggestIdx = -1
					return m, nil
				}
				// if slash command with exact match and manager filtering, Tab equivalent
				// let it fall through to submit handling if input is complete slash command without args
			}
		}

		// Main keybindings
		switch msg.String() {
		case "ctrl+c":
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit

		case "esc":
			if m.highlightQuery != "" {
				m.highlightQuery = ""
				m.highlightInput.SetValue("")
				return m, nil
			}
			if m.manager != managerNone {
				m.manager = managerNone
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
				m.input.Placeholder = "Type a word or short phrase to analyze... (Alt+Enter to submit)"
			} else {
				m.mode = ModeQuick
				m.input.Placeholder = "Type or paste text to translate... (Alt+Enter to submit)"
			}
			m.result = ""
			m.analysisResult = nil
			m.status = ""
			m.manager = managerNone
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

		case "ctrl+y":
			if m.result != "" {
				_ = clipboard.WriteAll(m.result)
				m.status = "copied"
			}
			return m, nil

		case "ctrl+s", "ctrl+enter", "alt+enter", "ctrl+j":
			return m.submit()
		case "enter":
			// Many terminals report Ctrl+Enter as a plain Enter. Slash commands
			// on the effective last line should still be executed, while
			// regular Enter inserts a newline.
			if strings.HasPrefix(effectiveSlashLine(m.input.Value()), "/") {
				// If suggestions are visible and a selection exists, autocomplete was already handled above.
				// Otherwise submit the slash command.
				kind, items, _, _ := m.suggestionItems()
				if kind == "command" && len(items) == 1 && m.suggestIdx == -1 {
					val := effectiveSlashLine(m.input.Value())
					if strings.EqualFold(val, items[0]) {
						return m.submit()
					}
				}
				if kind == "command" {
					val := effectiveSlashLine(m.input.Value())
					lower := strings.ToLower(val)
					for _, cmd := range slashCommands {
						if strings.ToLower(cmd.Name) == lower {
							return m.submit()
						}
						for _, alias := range cmd.Aliases {
							if strings.ToLower(alias) == lower {
								return m.submit()
							}
						}
					}
					// partial prefix -> autocomplete to first match on Enter
					if len(items) > 0 {
						m.applySuggestion(items[0])
						return m, nil
					}
				}
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
		return m, nil

	case modelsListedMsg:
		if msg.err == nil && len(msg.models) > 0 {
			if m.modelCache == nil {
				m.modelCache = make(map[string][]string)
			}
			m.modelCache[msg.provider] = msg.models
		}
		return m, nil
	}

	// default update input
	var cmd tea.Cmd
	prevVal := m.input.Value()
	m.input, cmd = m.input.Update(msg)
	// reset suggestion index when input text changes outside navigation
	if m.input.Value() != prevVal {
		// keep filter but reset index to allow ghost navigation
		// don't reset if user is navigating via up/down (already returned)
		if m.suggestIdx >= 0 {
			kind, items, _, _ := m.suggestionItems()
			_ = kind
			if m.suggestIdx >= len(items) {
				m.suggestIdx = -1
			}
		}
	}
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
	m.manager = managerNone
	m.suggestIdx = -1
	m.result = ""
	m.analysisResult = nil
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
	m.manager = managerNone
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
			m.status = "usage: /to <lang>"
			restoreRetained()
			return m, nil
		}
		lang := language.Normalize(args[0])
		if !language.IsKnown(lang) {
			m.status = "unknown language: " + args[0]
			restoreRetained()
			return m, nil
		}
		m.app.SetTarget(lang)
		for i, l := range m.langs {
			if l == lang {
				m.langIdx = i
			}
		}
		m.status = "target language set to " + strings.ToUpper(string(lang))

	case "/model":
		if len(args) == 0 {
			m.manager = managerModels
			m.status = "select a model with /model <name>  ·  Ctrl+T filter translation"
			restoreRetained()
			return m, m.fetchModelsCmd(m.app.Provider)
		}
		name := strings.Join(args, " ")
		m.app.SetModel(name)
		m.status = "model set to " + name

	case "/provider":
		if len(args) == 0 {
			m.manager = managerProviders
			m.status = "select a provider with /provider <name>"
			restoreRetained()
			return m, nil
		}
		name := args[0]
		// Inline key paste: "/provider openai sk-..." sets key directly and connects.
		if len(args) >= 2 && config.ProviderRequiresKey(name) {
			inlineKey := strings.Join(args[1:], " ")
			inlineKey = strings.TrimSpace(inlineKey)
			if len(inlineKey) > 8 {
				if err := m.app.SetProviderAPIKey(name, inlineKey); err == nil {
					m.app.SetProvider(name)
					m.status = "provider " + name + " connected (key saved)"
					restoreRetained()
					return m, m.fetchModelsCmd(name)
				}
			}
		}
		// If provider requires API key and none configured, prompt securely.
		if config.ProviderRequiresKey(name) && !m.app.HasAPIKey(name) {
			m.pendingProvider = name
			m.showAPIKey = true
			m.apiKeyInput.SetValue("")
			m.apiKeyInput.Focus()
			m.apiKeyErr = ""
			m.status = "paste API key for " + name + " to connect"
			restoreRetained()
			return m, nil
		}
		m.app.SetProvider(name)
		m.status = "provider set to " + name
		restoreRetained()
		return m, m.fetchModelsCmd(name)

	case "/word", "/analyze", "/mode":
		if len(args) > 0 {
			m.mode = ModeAnalyze
			// if there was retained translation, args take precedence
			m.input.SetValue(strings.Join(args, " "))
			m.input.Placeholder = "Type a word or short phrase to analyze... (Alt+Enter to submit)"
			m.status = "word analysis mode"
		} else {
			if m.mode == ModeQuick {
				m.mode = ModeAnalyze
				m.input.Placeholder = "Type a word or short phrase to analyze... (Alt+Enter to submit)"
				m.status = "word analysis mode"
			} else {
				m.mode = ModeQuick
				m.input.Placeholder = "Type or paste text to translate... (Alt+Enter to submit)"
				m.status = "quick translation mode"
			}
			restoreRetained()
		}
		m.result = ""
		m.analysisResult = nil

	case "/quit", "/exit", "/q":
		return m, tea.Quit

	case "/help", "/?":
		m.status = "commands: /to <lang>  /model <name>  /provider <name>  /word [text]  /help"
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
	cardWidth := m.cardWidth()

	// 1. Overlays (Model input or Highlight input or API key)
	if m.showAPIKey {
		errLine := ""
		if m.apiKeyErr != "" {
			errLine = "\n" + m.styles.StatusDanger.Render("✗ "+m.apiKeyErr)
		}
		label := fmt.Sprintf("Provider \"%s\" requires an API key.", m.pendingProvider)
		hint := "Enter to connect securely  •  Esc to cancel  •  key is saved to config (0600)"
		content := m.apiKeyInput.View() + errLine
		return m.renderModal(
			"Connect Provider",
			label,
			content,
			hint,
		)
	}
	if m.showModel {
		return m.renderModal(
			"Configure Model",
			"Enter model identifier:",
			m.modelInput.View(),
			"Enter to save  •  Esc to cancel",
		)
	}
	if m.showHighlight {
		return m.renderModal(
			"Highlight Sections / Terms",
			"Enter section name, word, or pattern to highlight:",
			m.highlightInput.View(),
			"Enter to apply  •  Esc to dismiss",
		)
	}

	// 2. Header Bar
	headerBar := m.renderHeader(cardWidth)

	// 3. Input Section — fixed frame containing textarea + ghost + slash menu.
	// The frame has a constant inner height (14 lines) so the slash menu appears
	// neatly inside it without pushing the result card or causing layout jump.
	inputFrame := m.renderInputCard(cardWidth)

	// 4. Result Section Card
	resultCard := m.renderResultCard(cardWidth)

	// 5. Status Line
	statusLine := m.renderStatusLine()

	// 6. Help / Keybindings Footer — hidden when idle preview is shown inside
	// the fixed input frame to avoid duplication (preview already says
	// "Type / for commands | Alt+Enter ...").
	footer := m.renderFooter(cardWidth)
	if kind, _, _, _ := m.suggestionItems(); kind == "" && m.manager == managerNone {
		footer = ""
	}

	// Assemble Main Content — input frame is fixed height, so no shift when "/" is typed.
	var b strings.Builder
	b.WriteString(headerBar)
	b.WriteString("\n\n")
	b.WriteString(inputFrame)
	b.WriteString("\n\n")
	b.WriteString(resultCard)
	if statusLine != "" {
		b.WriteString("\n")
		b.WriteString(statusLine)
	}
	b.WriteString("\n\n")
	b.WriteString(footer)

	content := b.String()

	// Center horizontally and vertically
	centered := lipgloss.PlaceHorizontal(width, lipgloss.Center, content)
	if m.height > 0 {
		centered = lipgloss.PlaceVertical(m.height, lipgloss.Center, centered)
	}
	return centered
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

	src := strings.ToUpper(string(m.app.SourceLang))
	if src == "" || src == "AUTO" {
		src = "AUTO"
	}
	tgt := strings.ToUpper(string(m.app.TargetLang))
	langPill := m.styles.BadgePill.Render(fmt.Sprintf("%s → %s", src, tgt))

	modelName := m.app.Model
	if len(modelName) > 16 {
		modelName = modelName[:14] + "…"
	}
	providerName := m.app.Provider
	modelPill := m.styles.BadgePill.Render(fmt.Sprintf("%s · %s", providerName, modelName))

	left := lipgloss.JoinHorizontal(lipgloss.Center, logo, "   ", modeCluster)
	right := lipgloss.JoinHorizontal(lipgloss.Center, langPill, "  ", modelPill)

	topRow := lipgloss.JoinHorizontal(lipgloss.Center, left, "   ", right)
	lineWidth := lipgloss.Width(topRow)
	if lineWidth < cardWidth {
		lineWidth = cardWidth
	}
	divider := m.styles.SubtleLine.Render(strings.Repeat("─", lineWidth))

	return topRow + "\n" + divider
}

func (m Model) renderInputCard(cardWidth int) string {
	modeLabel := "INPUT"
	if m.mode == ModeAnalyze {
		modeLabel = "WORD / PHRASE"
	}
	title := m.styles.BoxLabel.Render(modeLabel)

	charCount := len(strings.TrimSpace(m.input.Value()))
	metaText := ""
	if charCount > 0 {
		metaText = m.styles.BoxMeta.Render(fmt.Sprintf("%d chars", charCount))
	}

	headerRow := title
	if metaText != "" {
		spacing := cardWidth - lipgloss.Width(title) - lipgloss.Width(metaText) - 4
		if spacing > 0 {
			headerRow = title + strings.Repeat(" ", spacing) + metaText
		}
	}

	// ghost line: reserve 1 line inside the fixed frame to avoid jump;
	// show dim completion only when slash prefix is present.
	ghost := m.ghostText()
	ghostLine := ""
	hasGhost := false
	if ghost != "" && strings.HasPrefix(strings.TrimSpace(m.input.Value()), "/") {
		ghostLine = m.styles.Ghost.Render("  ↳ "+strings.TrimSpace(m.input.Value())+ghost) + "  " + m.styles.BoxMeta.Render("Tab to accept")
		hasGhost = true
	}

	// Fixed frame: header(1) + textarea(4) + ghost(1) + divider(1) + suggestions(7) = 14 inner lines.
	// We build the inner content and pad to fixedInputFrameInnerLines so the outer
	// result card never shifts when the slash menu appears/disappears.
	divider := m.styles.SubtleLine.Render(strings.Repeat("─", cardWidth-4))
	suggestionBlock := m.renderSuggestionInline(cardWidth)

	// Assemble inner content (without outer border) with fixed height.
	// textarea View is typically 4 lines; we treat it as one block.
	var parts []string
	parts = append(parts, headerRow)
	parts = append(parts, m.input.View())
	if hasGhost {
		parts = append(parts, ghostLine)
	} else {
		// reserve the ghost line as empty to keep frame height stable
		parts = append(parts, "")
	}
	parts = append(parts, divider)
	// suggestionBlock is already 7 lines joined; split and append
	suggestionLines := strings.Split(suggestionBlock, "\n")
	parts = append(parts, suggestionLines...)

	// Join and pad to fixed height
	inner := strings.Join(parts, "\n")
	// Count inner lines (textarea block may contain newlines already)
	innerLines := strings.Split(inner, "\n")
	// textarea's View already counts as 4, but Join added correct newlines.
	// Pad bottom with empty lines to reach fixed height if needed (should already be 14)
	for len(innerLines) < fixedInputFrameInnerLines {
		innerLines = append(innerLines, "")
	}
	if len(innerLines) > fixedInputFrameInnerLines {
		innerLines = innerLines[:fixedInputFrameInnerLines]
	}
	innerPadded := strings.Join(innerLines, "\n")

	boxStyle := m.styles.CardBoxFocus.Width(cardWidth)
	return boxStyle.Render(innerPadded)
}

// renderSuggestionInline returns a fixed-height (7 lines) block for the input frame.
// When no slash/manager is active it returns blank lines with a subtle placeholder,
// so the frame height stays constant and the menu appears neatly inside.
func (m Model) renderSuggestionInline(cardWidth int) string {
	// manager has priority when slash is not active
	if m.manager != managerNone {
		// reuse arg suggestions but render inline without outer box
		var lines []string
		switch m.manager {
		case managerProviders:
			lines = m.suggestionInlineLinesForArg("/provider", "", cardWidth)
		case managerModels:
			lines = m.suggestionInlineLinesForArg("/model", "", cardWidth)
		}
		// ensure fixed height (maxSuggestionRows + 2)
		for len(lines) < maxSuggestionRows+2 {
			lines = append(lines, "")
		}
		if len(lines) > maxSuggestionRows+2 {
			lines = lines[:maxSuggestionRows+2]
		}
		return strings.Join(lines, "\n")
	}

	kind, _, _, _ := m.suggestionItems()
	if kind == "" {
		// idle: no hidden commands — just a subtle preview in the reserved gap.
		// This keeps the fixed frame height stable (7 lines) while avoiding an
		// empty gap; the same text is hidden from the footer to avoid duplication.
		preview := m.styles.BoxMeta.Render("Type / for commands  |  Alt+Enter to submit  |  Tab switch mode  •  Esc clear")
		lines := make([]string, maxSuggestionRows+2)
		for i := range lines {
			lines[i] = ""
		}
		// center preview vertically in the 7-line block
		mid := (len(lines) - 1) / 2
		lines[mid] = preview
		return strings.Join(lines, "\n")
	}

	// active slash suggestions: render inline without outer SuggestBox border
	var lines []string
	if cmd, arg, ok := slashArgQuery(m.input.Value()); ok {
		lines = m.suggestionInlineLinesForArg(cmd, arg, cardWidth)
	} else {
		lines = m.suggestionInlineLinesForCommands(cardWidth)
	}
	for len(lines) < maxSuggestionRows+2 {
		lines = append(lines, "")
	}
	if len(lines) > maxSuggestionRows+2 {
		lines = lines[:maxSuggestionRows+2]
	}
	return strings.Join(lines, "\n")
}

func (m Model) suggestionInlineLinesForCommands(cardWidth int) []string {
	suggestions := slashSuggestionsFor(m.input.Value())
	title := m.styles.BoxLabel.Render("COMMANDS")
	lines := []string{title}
	visible := suggestions
	if len(visible) > maxSuggestionRows {
		visible = visible[:maxSuggestionRows]
	}
	for i, cmd := range visible {
		isSelected := i == m.suggestIdx
		if isSelected {
			lines = append(lines, m.styles.SuggestActive.Render(" ▸ "+cmd.Usage+"  "+cmd.Description))
		} else {
			lines = append(lines, "  "+m.styles.SuggestIdle.Render(cmd.Usage)+"  "+m.styles.BoxMeta.Render(cmd.Description))
		}
	}
	for len(lines) < maxSuggestionRows+1 {
		lines = append(lines, "")
	}
	if len(suggestions) > maxSuggestionRows {
		lines = append(lines, m.styles.BoxMeta.Render(fmt.Sprintf("  +%d more — keep typing to filter", len(suggestions)-maxSuggestionRows)))
	} else {
		lines = append(lines, m.styles.BoxMeta.Render("  ↑↓ navigate · Tab/Enter accept · Esc dismiss"))
	}
	return lines
}

func (m Model) suggestionInlineLinesForArg(cmd, arg string, cardWidth int) []string {
	var title string
	var names []string
	var hint string
	switch cmd {
	case "/provider":
		title = "PROVIDERS"
		names = filterByPrefix(m.providerNames(), arg)
		hint = "↑↓ navigate · Tab/Enter select · Esc dismiss"
	case "/model":
		title = "MODELS · " + strings.ToUpper(m.app.Provider)
		var all []string
		if m.filterExcellent {
			all = m.app.ModelsForProviderFiltered(m.app.Provider, true)
			cached := m.modelsForProvider(m.app.Provider)
			seen := map[string]bool{}
			for _, v := range all {
				seen[v] = true
			}
			for _, v := range cached {
				if app.IsTranslationExcellent(v) && !seen[v] {
					all = append(all, v)
				}
			}
			sort.Strings(all)
			var ex, other []string
			for _, v := range all {
				if app.IsTranslationExcellent(v) {
					ex = append(ex, v)
				} else {
					other = append(other, v)
				}
			}
			sort.Strings(ex)
			sort.Strings(other)
			all = append(ex, other...)
		} else {
			all = m.modelsForProvider(m.app.Provider)
			var ex, other []string
			for _, v := range all {
				if app.IsTranslationExcellent(v) {
					ex = append(ex, v)
				} else {
					other = append(other, v)
				}
			}
			sort.Strings(ex)
			sort.Strings(other)
			all = append(ex, other...)
		}
		names = filterByPrefix(all, arg)
		if m.filterExcellent {
			hint = "showing translation-optimized · Ctrl+T show all · ↑↓ Tab/Enter"
		} else {
			hint = "✦ translation-optimized first · Ctrl+T filter · ↑↓ Tab/Enter"
		}
	default:
		return []string{}
	}
	lines := []string{m.styles.BoxLabel.Render(title)}
	if len(names) == 0 {
		lines = append(lines, "  "+m.styles.EmptyState.Render("no matches"))
		for len(lines) < maxSuggestionRows+1 {
			lines = append(lines, "")
		}
		lines = append(lines, m.styles.BoxMeta.Render("  "+hint))
		return lines
	}
	total := len(names)
	visible := names
	if len(visible) > maxSuggestionRows {
		start := 0
		if m.suggestIdx >= 0 {
			start = m.suggestIdx - maxSuggestionRows/2
			if start < 0 {
				start = 0
			}
			if start+maxSuggestionRows > len(visible) {
				start = len(visible) - maxSuggestionRows
			}
		}
		visible = visible[start : start+maxSuggestionRows]
	}
	for _, name := range visible {
		isSelected := false
		if m.suggestIdx >= 0 && m.suggestIdx < len(names) && names[m.suggestIdx] == name {
			isSelected = true
		}
		marker := "  "
		active := name == m.app.Provider
		if cmd == "/model" {
			active = name == m.app.Model
		}
		if active {
			marker = "● "
		}
		if isSelected {
			marker = "▸ "
		}
		line := "  " + marker + name
		if app.IsTranslationExcellent(name) {
			line += " " + m.styles.BadgeExcellent.Render("✦")
		}
		if active {
			line += "  " + m.styles.BoxMeta.Render("(active)")
		}
		if isSelected {
			line = m.styles.SuggestActive.Render(line)
		}
		lines = append(lines, line)
	}
	for len(lines) < maxSuggestionRows+1 {
		lines = append(lines, "")
	}
	if total > maxSuggestionRows {
		lines = append(lines, m.styles.BoxMeta.Render(fmt.Sprintf("  %d/%d · %s", len(visible), total, hint)))
	} else {
		lines = append(lines, m.styles.BoxMeta.Render("  "+hint))
	}
	return lines
}

//nolint:unused // kept for backward compatibility; View now uses fixed-frame inline rendering
func (m Model) renderSlashSuggestions(cardWidth int) string {
	if cmd, arg, ok := slashArgQuery(m.input.Value()); ok {
		return m.renderArgSuggestions(cardWidth, cmd, arg)
	}

	suggestions := slashSuggestionsFor(m.input.Value())
	if len(suggestions) == 0 {
		return ""
	}

	title := m.styles.BoxLabel.Render("COMMANDS")
	lines := []string{title}
	visible := suggestions
	if len(visible) > maxSuggestionRows {
		visible = visible[:maxSuggestionRows]
	}
	for i, cmd := range visible {
		isSelected := i == m.suggestIdx
		usage := cmd.Usage
		desc := cmd.Description
		var line string
		if isSelected {
			line = m.styles.SuggestActive.Render(" ▸ " + usage + "  " + desc)
		} else {
			line = "  " + m.styles.SuggestIdle.Render(usage) + "  " + m.styles.BoxMeta.Render(desc)
		}
		lines = append(lines, line)
	}
	// pad to fixed height to avoid erratic jumping when filtering
	for len(lines) < maxSuggestionRows+1 {
		lines = append(lines, "")
	}
	if len(suggestions) > maxSuggestionRows {
		lines = append(lines, m.styles.BoxMeta.Render(fmt.Sprintf("  +%d more — keep typing to filter", len(suggestions)-maxSuggestionRows)))
	} else {
		lines = append(lines, m.styles.BoxMeta.Render("  ↑↓ navigate · Tab/Enter accept · Esc dismiss"))
	}

	boxStyle := m.styles.SuggestBox.Width(cardWidth)
	return boxStyle.Render(strings.Join(lines, "\n"))
}

//nolint:unused // inline version is used inside fixed frame
func (m Model) renderArgSuggestions(cardWidth int, cmd, arg string) string {
	var title string
	var names []string
	var hint string

	switch cmd {
	case "/provider":
		title = "PROVIDERS"
		names = filterByPrefix(m.providerNames(), arg)
		hint = "↑↓ navigate · Tab/Enter select · Esc dismiss"
	case "/model":
		title = "MODELS · " + strings.ToUpper(m.app.Provider)
		var all []string
		if m.filterExcellent {
			all = m.app.ModelsForProviderFiltered(m.app.Provider, true)
			// include cached
			cached := m.modelsForProvider(m.app.Provider)
			seen := map[string]bool{}
			for _, v := range all {
				seen[v] = true
			}
			for _, v := range cached {
				if app.IsTranslationExcellent(v) && !seen[v] {
					all = append(all, v)
				}
			}
			sort.Strings(all)
			var ex, other []string
			for _, v := range all {
				if app.IsTranslationExcellent(v) {
					ex = append(ex, v)
				} else {
					other = append(other, v)
				}
			}
			sort.Strings(ex)
			sort.Strings(other)
			all = append(ex, other...)
		} else {
			all = m.modelsForProvider(m.app.Provider)
			var ex, other []string
			for _, v := range all {
				if app.IsTranslationExcellent(v) {
					ex = append(ex, v)
				} else {
					other = append(other, v)
				}
			}
			sort.Strings(ex)
			sort.Strings(other)
			all = append(ex, other...)
		}
		names = filterByPrefix(all, arg)
		if m.filterExcellent {
			hint = "showing translation-optimized · Ctrl+T show all · ↑↓ Tab/Enter"
		} else {
			hint = "✦ translation-optimized first · Ctrl+T filter · ↑↓ Tab/Enter"
		}
	default:
		return ""
	}

	lines := []string{m.styles.BoxLabel.Render(title)}
	if len(names) == 0 {
		lines = append(lines, "  "+m.styles.EmptyState.Render("no matches"))
		// pad to fixed height
		for len(lines) < maxSuggestionRows+1 {
			lines = append(lines, "")
		}
		lines = append(lines, m.styles.BoxMeta.Render("  "+hint))
		boxStyle := m.styles.SuggestBox.Width(cardWidth)
		return boxStyle.Render(strings.Join(lines, "\n"))
	}
	// limit visible rows
	total := len(names)
	visible := names
	if len(visible) > maxSuggestionRows {
		// scroll window around selected index
		start := 0
		if m.suggestIdx >= 0 {
			start = m.suggestIdx - maxSuggestionRows/2
			if start < 0 {
				start = 0
			}
			if start+maxSuggestionRows > len(visible) {
				start = len(visible) - maxSuggestionRows
			}
		}
		visible = visible[start : start+maxSuggestionRows]
	}
	for _, name := range visible {
		// find original index for selection highlight
		isSelected := false
		if m.suggestIdx >= 0 && m.suggestIdx < len(names) && names[m.suggestIdx] == name {
			isSelected = true
		}
		marker := "  "
		active := name == m.app.Provider
		if cmd == "/model" {
			active = name == m.app.Model
		}
		if active {
			marker = "● "
		}
		if isSelected {
			marker = "▸ "
		}
		line := "  " + marker + name
		if app.IsTranslationExcellent(name) {
			line += " " + m.styles.BadgeExcellent.Render("✦")
		}
		if active {
			line += "  " + m.styles.BoxMeta.Render("(active)")
		}
		if isSelected {
			line = m.styles.SuggestActive.Render(line)
		}
		lines = append(lines, line)
	}
	// pad to fixed height to keep layout stable when filtering reduces count
	for len(lines) < maxSuggestionRows+1 {
		lines = append(lines, "")
	}
	if total > maxSuggestionRows {
		lines = append(lines, m.styles.BoxMeta.Render(fmt.Sprintf("  %d/%d · %s", len(visible), total, hint)))
	} else {
		lines = append(lines, m.styles.BoxMeta.Render("  "+hint))
	}

	boxStyle := m.styles.SuggestBox.Width(cardWidth)
	return boxStyle.Render(strings.Join(lines, "\n"))
}

//nolint:unused
func (m Model) renderManager(cardWidth int) string {
	switch m.manager {
	case managerProviders:
		return m.renderArgSuggestions(cardWidth, "/provider", "")
	case managerModels:
		return m.renderArgSuggestions(cardWidth, "/model", "")
	}
	return ""
}

func (m Model) renderResultCard(cardWidth int) string {
	resLabel := "TRANSLATION"
	if m.mode == ModeAnalyze {
		resLabel = "ANALYSIS"
	}
	title := m.styles.BoxLabel.Render(resLabel)

	var highlightBadge string
	if m.highlightQuery != "" {
		highlightBadge = m.styles.HighlightMatch.Render(fmt.Sprintf(" [highlight: %s] ", m.highlightQuery))
	}

	headerRow := title
	if highlightBadge != "" {
		spacing := cardWidth - lipgloss.Width(title) - lipgloss.Width(highlightBadge) - 4
		if spacing > 0 {
			headerRow = title + strings.Repeat(" ", spacing) + highlightBadge
		} else {
			headerRow = title + " " + highlightBadge
		}
	}

	var content string
	if m.loading {
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
	} else if m.mode == ModeAnalyze && m.analysisResult != nil {
		content = RenderAnalysisStructured(*m.analysisResult, m.highlightQuery, m.styles, cardWidth)
	} else if m.result != "" {
		content = RenderHighlightedText(m.result, m.highlightQuery, m.styles)
	} else {
		content = m.styles.EmptyState.Render("Results will appear here. Press Alt+Enter to submit.")
	}

	boxStyle := m.styles.CardBox.Width(cardWidth)
	return boxStyle.Render(headerRow + "\n" + content)
}

func (m Model) renderStatusLine() string {
	if m.status == "" || m.loading {
		return ""
	}
	noticePrefixes := []string{
		"select a ", "target language set to ", "model set to ", "provider set to ",
		"word analysis mode", "quick translation mode", "filtering", "showing all",
		"provider ", "paste API",
	}
	for _, prefix := range noticePrefixes {
		if strings.HasPrefix(m.status, prefix) {
			return m.styles.BoxMeta.Render(m.status)
		}
	}
	switch m.status {
	case "done":
		return m.styles.StatusSuccess.Render("✓ Done")
	case "copied":
		return m.styles.StatusSuccess.Render("✓ Copied to clipboard")
	case "cancelled":
		return m.styles.StatusWarning.Render("⊘ Request cancelled")
	default:
		return m.styles.StatusDanger.Render("✗ " + m.status)
	}
}

func (m Model) renderFooter(cardWidth int) string {
	footerText := "Type / for commands  |  Alt+Enter to submit  |  Tab switch mode  •  Esc clear"
	return lipgloss.NewStyle().Width(cardWidth).Align(lipgloss.Center).Foreground(lipgloss.Color(ColorTextMuted)).Render(footerText)
}

func (m Model) renderModal(title, label, inputView, hint string) string {
	titleStyled := m.styles.ModalTitle.Render(title)
	labelStyled := m.styles.BoxLabel.Render(label)
	hintStyled := m.styles.ModalHint.Render(hint)

	modalContent := fmt.Sprintf("%s\n\n%s\n%s\n\n%s", titleStyled, labelStyled, inputView, hintStyled)
	modalBox := m.styles.ModalBox.Width(56).Render(modalContent)

	width := m.width
	if width == 0 {
		width = 80
	}
	height := m.height
	if height == 0 {
		height = 24
	}

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
