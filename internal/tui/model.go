package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

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

type Model struct {
	app *app.App

	mode          Mode
	width, height int

	input   textarea.Model
	result  string
	status  string
	loading bool

	cancel context.CancelFunc
	chunks <-chan string
	errs   <-chan error

	// model overlay
	modelInput textinput.Model
	showModel  bool

	// languages
	langs   []language.Language
	langIdx int

	styles Styles
}

func NewModel(a *app.App) Model {
	ta := textarea.New()
	ta.Placeholder = "Type text to translate..."
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.SetHeight(5)

	ti := textinput.New()
	ti.Placeholder = "model name"
	ti.CharLimit = 128

	langs := []language.Language{language.VI, language.EN, language.JA, language.KO, language.ZH, language.FR, language.ES, language.DE}
	// find current target
	idx := 0
	for i, l := range langs {
		if l == a.TargetLang {
			idx = i
		}
	}
	return Model{
		app:        a,
		mode:       ModeQuick,
		input:      ta,
		modelInput: ti,
		langs:      langs,
		langIdx:    idx,
		styles:     DefaultStyles(),
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(msg.Width - 4)
		return m, nil

	case tea.KeyMsg:
		// model overlay
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

		switch msg.String() {
		case "ctrl+c":
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		case "esc":
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
				m.input.Placeholder = "Type a word or phrase to analyze..."
			} else {
				m.mode = ModeQuick
				m.input.Placeholder = "Type text to translate..."
			}
			m.result = ""
			m.status = ""
			return m, nil
		case "ctrl+l":
			m.langIdx = (m.langIdx + 1) % len(m.langs)
			m.app.SetTarget(m.langs[m.langIdx])
			return m, nil
		case "ctrl+m":
			m.showModel = true
			m.modelInput.SetValue(m.app.Model)
			m.modelInput.Focus()
			return m, nil
		case "ctrl+y":
			if m.result != "" {
				_ = clipboard.WriteAll(m.result)
				m.status = "copied"
			}
			return m, nil
		case "ctrl+enter", "alt+enter":
			return m.submit()
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
		m.result = formatAnalysis(msg.res)
		m.status = "done"
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
	if m.loading {
		return m, nil
	}
	m.result = ""
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

	// analyze
	return m, func() tea.Msg {
		res, err := m.app.Analysis.Analyze(ctx, analysis.Request{
			Word:       text,
			SourceLang: m.app.SourceLang,
			TargetLang: m.app.TargetLang,
		})
		return analysisDoneMsg{res: res, err: err}
	}
}

func waitForChunk(ch <-chan string, errCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case c, ok := <-ch:
			if !ok {
				// channel closed; check err
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
	if m.width == 0 {
		m.width = 80
	}
	modeStr := "Quick"
	if m.mode == ModeAnalyze {
		modeStr = "Analyze"
	}
	header := m.styles.Header.Render(fmt.Sprintf("tnow  [%s]   %s → %s   %s   %s", modeStr, m.app.SourceLang, m.app.TargetLang, m.app.Model, m.app.Provider))

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")
	b.WriteString(m.styles.Label.Render("Input"))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(m.styles.Label.Render("Result"))
	b.WriteString("\n")
	b.WriteString(m.result)
	b.WriteString("\n\n")
	if m.status != "" {
		b.WriteString(m.styles.Status.Render(m.status))
		b.WriteString("\n")
	}
	if m.showModel {
		b.WriteString("\n")
		b.WriteString(m.styles.Label.Render("Model: "))
		b.WriteString(m.modelInput.View())
	}
	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("Enter newline  Ctrl+Enter submit  Tab mode  Ctrl+Y copy  Ctrl+L language  Ctrl+M model  Esc cancel  Ctrl+C quit"))
	return b.String()
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
