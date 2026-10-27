package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/andev0x/tnow/internal/analysis"
	"github.com/andev0x/tnow/internal/app"
	"github.com/andev0x/tnow/internal/config"
)

func TestHighlightMatches(t *testing.T) {
	s := DefaultStyles()

	tests := []struct {
		name     string
		text     string
		query    string
		contains []string
	}{
		{
			name:     "empty query returns text as-is",
			text:     "Hello world",
			query:    "",
			contains: []string{"Hello world"},
		},
		{
			name:     "case insensitive match",
			text:     "The Awkward silence was awkward.",
			query:    "awkward",
			contains: []string{"Awkward", "awkward"},
		},
		{
			name:     "no match",
			text:     "Simple sentence",
			query:    "zebra",
			contains: []string{"Simple sentence"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HighlightMatches(tt.text, tt.query, s)
			for _, expected := range tt.contains {
				if !strings.Contains(got, expected) {
					t.Errorf("HighlightMatches(%q, %q) does not contain %q, got: %q", tt.text, tt.query, expected, got)
				}
			}
		})
	}
}

func TestRenderHighlightedText(t *testing.T) {
	s := DefaultStyles()

	t.Run("renders inline markers", func(t *testing.T) {
		input := "This is ==highlighted== text with **bold** and `code`."
		rendered := RenderHighlightedText(input, "", s)

		if !strings.Contains(rendered, "highlighted") {
			t.Errorf("expected 'highlighted' in rendered output, got: %s", rendered)
		}
		if !strings.Contains(rendered, "bold") {
			t.Errorf("expected 'bold' in rendered output, got: %s", rendered)
		}
		if !strings.Contains(rendered, "code") {
			t.Errorf("expected 'code' in rendered output, got: %s", rendered)
		}
	})

	t.Run("renders headers and bullets", func(t *testing.T) {
		input := "### Section Title\n• Point one\n- Point two\nA -> B"
		rendered := RenderHighlightedText(input, "", s)

		if !strings.Contains(rendered, "Section Title") {
			t.Errorf("expected section title in output, got: %s", rendered)
		}
		if !strings.Contains(rendered, "Point one") {
			t.Errorf("expected bullet point in output, got: %s", rendered)
		}
		if !strings.Contains(rendered, "➜") {
			t.Errorf("expected arrow symbol in output, got: %s", rendered)
		}
	})

	t.Run("applies query highlight", func(t *testing.T) {
		input := "The deployment failed unexpectedly."
		rendered := RenderHighlightedText(input, "failed", s)

		if !strings.Contains(rendered, "failed") {
			t.Errorf("expected query in output, got: %s", rendered)
		}
	})
}

func TestRenderAnalysisStructured(t *testing.T) {
	s := DefaultStyles()

	res := analysis.Result{
		Word:         "awkward",
		Language:     "en",
		IPA:          "ˈɔːkwəd",
		PartOfSpeech: "adjective",
		Definitions: []analysis.Definition{
			{
				Meaning:      "causing or feeling embarrassed",
				Translations: []string{"ngượng ngùng", "lúng túng"},
			},
		},
		Examples: []analysis.Example{
			{
				Source:      "an awkward silence",
				Translation: "một sự im lặng ngượng ngùng",
			},
		},
	}

	rendered := RenderAnalysisStructured(res, "", s, 80)

	if !strings.Contains(rendered, "awkward") {
		t.Errorf("expected word 'awkward' in output, got: %s", rendered)
	}
	if !strings.Contains(rendered, "ˈɔːkwəd") {
		t.Errorf("expected IPA in output, got: %s", rendered)
	}
	if !strings.Contains(rendered, "ADJECTIVE") {
		t.Errorf("expected part of speech in output, got: %s", rendered)
	}
	if !strings.Contains(rendered, "MEANINGS") {
		t.Errorf("expected MEANINGS section header, got: %s", rendered)
	}
	if !strings.Contains(rendered, "ngượng ngùng") {
		t.Errorf("expected translations in output, got: %s", rendered)
	}
	if !strings.Contains(rendered, "EXAMPLES") {
		t.Errorf("expected EXAMPLES section header, got: %s", rendered)
	}
	if !strings.Contains(rendered, "an awkward silence") {
		t.Errorf("expected example source in output, got: %s", rendered)
	}

	// Test with query highlighting
	withQuery := RenderAnalysisStructured(res, "silence", s, 80)
	if !strings.Contains(withQuery, "silence") {
		t.Errorf("expected 'silence' in query highlighted output")
	}
}

func TestModelViewAndKeybindings(t *testing.T) {
	cfg := config.Default()
	cfg.TargetLang = "vi"
	a, err := app.New(cfg, "", "")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	m := NewModel(a)

	// Window size message
	tm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = tm.(Model)
	if m.width != 100 || m.height != 40 {
		t.Errorf("expected window size 100x40, got %dx%d", m.width, m.height)
	}

	// Initial view rendering
	view := m.View()
	if !strings.Contains(view, "tnow") {
		t.Errorf("expected view to contain app title 'tnow', got: %s", view)
	}
	if !strings.Contains(view, "Quick") {
		t.Errorf("expected view to contain 'Quick' mode badge")
	}
	if !strings.Contains(view, "INPUT") {
		t.Errorf("expected view to contain 'INPUT' label")
	}

	// Switch mode with Tab
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = tm.(Model)
	if m.mode != ModeAnalyze {
		t.Errorf("expected mode to switch to ModeAnalyze")
	}
	view = m.View()
	if !strings.Contains(view, "WORD / PHRASE") {
		t.Errorf("expected view to contain 'WORD / PHRASE' label in analyze mode")
	}

	// Switch back
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = tm.(Model)
	if m.mode != ModeQuick {
		t.Errorf("expected mode to switch back to ModeQuick")
	}

	// Toggle Highlight overlay with Ctrl+H
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	m = tm.(Model)
	if !m.showHighlight {
		t.Errorf("expected showHighlight to be true")
	}
	view = m.View()
	if !strings.Contains(view, "Highlight Sections / Terms") {
		t.Errorf("expected highlight modal in view, got: %s", view)
	}

	// Type highlight query and press enter
	m.highlightInput.SetValue("failure")
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(Model)
	if m.showHighlight {
		t.Errorf("expected showHighlight to be false after Enter")
	}
	if m.highlightQuery != "failure" {
		t.Errorf("expected highlightQuery to be 'failure', got: %s", m.highlightQuery)
	}

	// Clear highlight query with Esc
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(Model)
	if m.highlightQuery != "" {
		t.Errorf("expected highlightQuery to be cleared by Esc, got: %s", m.highlightQuery)
	}

	// Cycle language with Ctrl+L
	origLang := m.app.TargetLang
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	m = tm.(Model)
	if m.app.TargetLang == origLang {
		t.Errorf("expected target language to change with Ctrl+L")
	}

	// Toggle Model overlay with Alt+M / Ctrl+M
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}, Alt: true})
	m = tm.(Model)
	if !m.showModel {
		t.Errorf("expected showModel to be true")
	}
	view = m.View()
	if !strings.Contains(view, "Configure Model") {
		t.Errorf("expected model modal in view, got: %s", view)
	}
	// Dismiss model modal with Esc
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(Model)
	if m.showModel {
		t.Errorf("expected showModel to be false after Esc")
	}

	// Test spinner tick
	_, cmd := m.Update(spinner.TickMsg{})
	if cmd == nil {
		t.Errorf("expected spinner update to return a tick cmd")
	}
}
