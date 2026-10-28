package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/andev0x/tnow/internal/language"
	"github.com/andev0x/tnow/internal/runtime"
	"github.com/andev0x/tnow/internal/translation"
)

// blockingProvider holds Generate until the test releases it, simulating a
// provider whose HTTP response headers take a while to arrive.
type blockingProvider struct {
	release chan struct{}
}

func (p *blockingProvider) Name() string { return "blocking" }

func (p *blockingProvider) Generate(ctx context.Context, _ runtime.GenerationRequest) (runtime.GenerationResult, error) {
	select {
	case <-p.release:
		return runtime.GenerationResult{Text: "ok"}, nil
	case <-ctx.Done():
		return runtime.GenerationResult{}, ctx.Err()
	}
}

func TestAnalyzeLoadingOnlyUsesCentralSpinner(t *testing.T) {
	m := sizedModel(t, 80, 24)
	m.mode = ModeAnalyze
	m.loading = true
	m.status = "analyzing..."

	glyph := ansi.Strip(m.spinner.View())
	status, _, _ := m.footerStatus()
	if status != "" {
		t.Errorf("analyze footer should not repeat the loading indicator, got %q", status)
	}

	view := ansi.Strip(m.View())
	if count := strings.Count(view, glyph); count != 1 {
		t.Errorf("view contains %d spinners, want only the central one:\n%s", count, view)
	}
	if count := strings.Count(view, "analyzing..."); count != 1 {
		t.Errorf("view contains %d loading labels, want only the central one:\n%s", count, view)
	}
}

func TestAutoCopiedStatusShowsSingleCheck(t *testing.T) {
	m := sizedModel(t, 80, 24)
	m.status = "auto-copied"

	status, _, _ := m.footerStatus()
	if status != "✓ auto-copied" {
		t.Errorf("footer status = %q, want a single check mark", status)
	}
	view := ansi.Strip(m.View())
	if count := strings.Count(view, "✓ auto-copied"); count != 1 {
		t.Errorf("view shows %d auto-copied labels, want exactly one:\n%s", count, view)
	}
}

func TestQuickSubmitShowsSpinnerBeforeProviderResponds(t *testing.T) {
	p := &blockingProvider{release: make(chan struct{})}
	t.Cleanup(func() { close(p.release) })

	rt := runtime.New(map[string]runtime.Provider{"blocking": p}, "blocking")
	m := sizedModel(t, 80, 24)
	m.app.Translation = translation.NewEngine(rt, "blocking", "test-model")
	m.app.SourceLang = language.Auto
	m.app.TargetLang = language.VI
	m.input.SetValue("hello")
	m.input.CursorEnd()

	type submitted struct {
		model tea.Model
		cmd   tea.Cmd
	}
	done := make(chan submitted, 1)
	go func() {
		tm, cmd := m.submit()
		done <- submitted{model: tm, cmd: cmd}
	}()

	select {
	case got := <-done:
		m = got.model.(Model)
		if got.cmd == nil {
			t.Fatal("expected a command that starts the translation stream")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("submit blocked until the provider responded instead of showing progress immediately")
	}

	if !m.loading {
		t.Fatal("expected loading state immediately after submit")
	}
	assertOnlyFooterSpinner(t, m)
}

func TestQuickStreamingKeepsSpinnerInFooter(t *testing.T) {
	m := sizedModel(t, 80, 24)
	m.mode = ModeQuick
	m.loading = true
	m.status = "streaming..."
	// Overflowing output used to put a spinner in the result card meta.
	m.result = strings.Repeat("một bản dịch dài ", 40)

	assertOnlyFooterSpinner(t, m)
}

// assertOnlyFooterSpinner verifies that Quick mode has exactly one spinner and
// that it sits on the bottom status row.
func assertOnlyFooterSpinner(t *testing.T, m Model) {
	t.Helper()
	glyph := ansi.Strip(m.spinner.View())
	lines := plainLines(m.View())
	count := 0
	for i, line := range lines {
		n := strings.Count(line, glyph)
		count += n
		if n > 0 && i != len(lines)-1 {
			t.Errorf("spinner found outside the footer on row %d: %q", i, line)
		}
	}
	if count != 1 {
		t.Errorf("found %d spinners, want exactly one in the footer", count)
	}
	if len(lines) == 0 || !strings.Contains(lines[len(lines)-1], glyph) {
		t.Error("footer is missing the spinner")
	}
}

func TestStreamStartWiresChunkMessages(t *testing.T) {
	m := sizedModel(t, 80, 24)
	m.app.Config.AutoCopy = false
	m.loading = true
	m.streamID = 1

	chunks := make(chan string, 1)
	errs := make(chan error, 1)
	tm, cmd := m.Update(streamStartMsg{id: 1, chunks: chunks, errs: errs})
	m = tm.(Model)
	if !m.loading {
		t.Fatal("starting a stream should keep the loading state")
	}
	if cmd == nil {
		t.Fatal("expected a command waiting for the first chunk")
	}

	chunks <- "ok"
	tm, cmd = m.Update(cmd())
	m = tm.(Model)
	if m.result != "ok" {
		t.Fatalf("result = %q, want ok", m.result)
	}
	if cmd == nil {
		t.Fatal("expected a command waiting for the next chunk")
	}

	close(chunks)
	tm, _ = m.Update(cmd())
	m = tm.(Model)
	if m.loading {
		t.Fatal("stream completion should clear the loading state")
	}
	if m.status != "done" {
		t.Fatalf("status = %q, want done", m.status)
	}
}
