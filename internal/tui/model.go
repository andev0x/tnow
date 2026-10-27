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
	{Name: "/quick", Usage: "/quick", Description: "switch to quick translation mode"},
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

// slashSuggestionsFor returns the commands matching the current single-line
// slash input. Suggestions disappear once arguments start being typed.
func slashSuggestionsFor(value string) []slashCommand {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "/") || strings.Contains(value, "\n") {
		return nil
	}
	fields := strings.Fields(value)
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
// the optional argument used to filter the list.
func slashArgQuery(value string) (cmd, arg string, ok bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "/") || strings.Contains(value, "\n") {
		return "", "", false
	}
	if i := strings.IndexAny(value, " \t"); i >= 0 {
		cmd = strings.ToLower(value[:i])
		arg = strings.TrimSpace(value[i+1:])
	} else {
		cmd = strings.ToLower(value)
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

	// languages
	langs   []language.Language
	langIdx int

	// provider/model manager
	manager    managerKind
	modelCache map[string][]string

	styles Styles
}

func NewModel(a *app.App) Model {
	styles := DefaultStyles()

	ta := textarea.New()
	ta.Placeholder = "Type or paste text to translate... (Ctrl+Enter to submit)"
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
		app:            a,
		mode:           ModeQuick,
		input:          ta,
		modelInput:     ti,
		highlightInput: hi,
		spinner:        sp,
		langs:          langs,
		langIdx:        idx,
		modelCache:     make(map[string][]string),
		styles:         styles,
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
				m.input.Placeholder = "Type a word or short phrase to analyze... (Ctrl+Enter to submit)"
			} else {
				m.mode = ModeQuick
				m.input.Placeholder = "Type or paste text to translate... (Ctrl+Enter to submit)"
			}
			m.result = ""
			m.analysisResult = nil
			m.status = ""
			m.manager = managerNone
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
			// Many terminals report Ctrl+Enter as a plain Enter. A slash
			// command is never multi-line input, so execute it here while
			// leaving regular Enter as a newline insertion.
			if strings.HasPrefix(strings.TrimSpace(m.input.Value()), "/") {
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
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) submit() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		return m, nil
	}
	// Slash commands are local state changes, so they should still work
	// while a previous translation/analysis request is in flight.
	if strings.HasPrefix(text, "/") {
		return m.handleSlashCommand(text)
	}
	if m.loading {
		return m, nil
	}
	m.manager = managerNone
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
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) == 0 {
		return m, nil
	}
	cmd := strings.ToLower(fields[0])
	args := fields[1:]
	m.manager = managerNone

	switch cmd {
	case "/to", "/lang", "/language", "/target":
		if len(args) == 0 {
			m.status = "usage: /to <lang>"
			m.input.SetValue("")
			return m, nil
		}
		lang := language.Normalize(args[0])
		if !language.IsKnown(lang) {
			m.status = "unknown language: " + args[0]
			m.input.SetValue("")
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
			m.status = "select a model with /model <name>"
			m.input.SetValue("")
			return m, m.fetchModelsCmd(m.app.Provider)
		}
		name := strings.Join(args, " ")
		m.app.SetModel(name)
		m.status = "model set to " + name

	case "/provider":
		if len(args) == 0 {
			m.manager = managerProviders
			m.status = "select a provider with /provider <name>"
			m.input.SetValue("")
			return m, nil
		}
		name := args[0]
		m.app.SetProvider(name)
		m.status = "provider set to " + name
		m.input.SetValue("")
		return m, m.fetchModelsCmd(name)

	case "/word", "/analyze", "/mode":
		if len(args) > 0 {
			m.mode = ModeAnalyze
			m.input.SetValue(strings.Join(args, " "))
			m.input.Placeholder = "Type a word or short phrase to analyze... (Ctrl+Enter to submit)"
			m.status = "word analysis mode"
		} else {
			if m.mode == ModeQuick {
				m.mode = ModeAnalyze
				m.input.Placeholder = "Type a word or short phrase to analyze... (Ctrl+Enter to submit)"
				m.status = "word analysis mode"
			} else {
				m.mode = ModeQuick
				m.input.Placeholder = "Type or paste text to translate... (Ctrl+Enter to submit)"
				m.status = "quick translation mode"
			}
			m.input.SetValue("")
		}
		m.result = ""
		m.analysisResult = nil

	case "/quick":
		m.mode = ModeQuick
		m.input.Placeholder = "Type or paste text to translate... (Ctrl+Enter to submit)"
		m.status = "quick translation mode"
		m.result = ""
		m.analysisResult = nil
		m.input.SetValue("")

	case "/help", "/?":
		m.status = "commands: /to <lang>  /model <name>  /provider <name>  /word [text]  /help"
		m.input.SetValue("")

	default:
		m.status = "unknown command: " + cmd
		m.input.SetValue("")
		return m, nil
	}

	// clear input for commands that did not explicitly set it
	if cmd != "/word" || len(args) == 0 {
		// for /word with text we keep text in input; otherwise ensure cleared
		if !(cmd == "/word" && len(args) > 0) {
			m.input.SetValue("")
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

	// 1. Overlays (Model input or Highlight input)
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

	// 3. Input Section Card
	inputCard := m.renderInputCard(cardWidth)

	// 3b. Slash command suggestions or the active provider/model manager
	suggestions := m.renderSlashSuggestions(cardWidth)
	manager := ""
	if suggestions == "" {
		manager = m.renderManager(cardWidth)
	}

	// 4. Result Section Card
	resultCard := m.renderResultCard(cardWidth)

	// 5. Status Line
	statusLine := m.renderStatusLine()

	// 6. Help / Keybindings Footer
	footer := m.renderFooter(cardWidth)

	// Assemble Main Content
	var b strings.Builder
	b.WriteString(headerBar)
	b.WriteString("\n\n")
	b.WriteString(inputCard)
	if suggestions != "" {
		b.WriteString("\n")
		b.WriteString(suggestions)
	} else if manager != "" {
		b.WriteString("\n")
		b.WriteString(manager)
	}
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

	boxStyle := m.styles.CardBoxFocus.Copy().Width(cardWidth)
	cardContent := headerRow + "\n" + m.input.View()
	return boxStyle.Render(cardContent)
}

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
	for _, cmd := range suggestions {
		usage := m.styles.ModalTitle.Render(cmd.Usage)
		desc := m.styles.BoxMeta.Render(cmd.Description)
		lines = append(lines, "  "+usage+"  "+desc)
	}

	boxStyle := m.styles.CardBox.Copy().Width(cardWidth)
	return boxStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) renderArgSuggestions(cardWidth int, cmd, arg string) string {
	var title string
	var names []string
	var hint string

	switch cmd {
	case "/provider":
		title = "PROVIDERS"
		names = filterByPrefix(m.providerNames(), arg)
		hint = "/provider <name>"
	case "/model":
		title = "MODELS · " + strings.ToUpper(m.app.Provider)
		names = filterByPrefix(m.modelsForProvider(m.app.Provider), arg)
		hint = "/model <name>"
	default:
		return ""
	}

	lines := []string{m.styles.BoxLabel.Render(title)}
	if len(names) == 0 {
		lines = append(lines, "  "+m.styles.EmptyState.Render("no matches"))
	}
	for _, name := range names {
		marker := "  "
		active := name == m.app.Provider
		if cmd == "/model" {
			active = name == m.app.Model
		}
		if active {
			marker = "● "
		}
		line := "  " + marker + name
		if cmd == "/provider" {
			if models := m.modelsForProvider(name); len(models) > 0 {
				line += "  " + m.styles.BoxMeta.Render("· "+strings.Join(models, ", "))
			}
		}
		if active {
			line += "  " + m.styles.BoxMeta.Render("(active)")
		}
		lines = append(lines, line)
	}
	lines = append(lines, m.styles.BoxMeta.Render("  "+hint))

	boxStyle := m.styles.CardBox.Copy().Width(cardWidth)
	return boxStyle.Render(strings.Join(lines, "\n"))
}

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
		content = m.styles.EmptyState.Render("Results will appear here. Press Ctrl+Enter to submit.")
	}

	boxStyle := m.styles.CardBox.Copy().Width(cardWidth)
	return boxStyle.Render(headerRow + "\n" + content)
}

func (m Model) renderStatusLine() string {
	if m.status == "" || m.loading {
		return ""
	}
	noticePrefixes := []string{
		"select a ", "target language set to ", "model set to ", "provider set to ",
		"word analysis mode", "quick translation mode",
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
	footerText := "Type /help for commands  |  Ctrl+Enter to submit  |  Esc to quit"
	return lipgloss.NewStyle().Width(cardWidth).Align(lipgloss.Center).Foreground(lipgloss.Color(ColorTextMuted)).Render(footerText)
}

func (m Model) renderModal(title, label, inputView, hint string) string {
	titleStyled := m.styles.ModalTitle.Render(title)
	labelStyled := m.styles.BoxLabel.Render(label)
	hintStyled := m.styles.ModalHint.Render(hint)

	modalContent := fmt.Sprintf("%s\n\n%s\n%s\n\n%s", titleStyled, labelStyled, inputView, hintStyled)
	modalBox := m.styles.ModalBox.Copy().Width(56).Render(modalContent)

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
