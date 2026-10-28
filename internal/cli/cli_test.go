package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andev0x/tnow/internal/analysis"
	"github.com/andev0x/tnow/internal/app"
	"github.com/andev0x/tnow/internal/language"
	"github.com/andev0x/tnow/internal/runtime"
	"github.com/andev0x/tnow/internal/translation"
)

type mockProvider struct {
	res runtime.GenerationResult
	err error
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Generate(ctx context.Context, req runtime.GenerationRequest) (runtime.GenerationResult, error) {
	return m.res, m.err
}

func newTestApp(res runtime.GenerationResult, err error) *app.App {
	mp := &mockProvider{res: res, err: err}
	rt := runtime.New(map[string]runtime.Provider{"mock": mp}, "mock")
	a := &app.App{
		Runtime:    rt,
		SourceLang: language.Auto,
		TargetLang: language.VI,
		Provider:   "mock",
		Model:      "m",
	}
	a.Translation = translation.NewEngine(rt, "mock", "m")
	a.Analysis = analysis.NewEngine(rt, "mock", "m")
	return a
}

func TestCLIStdout(t *testing.T) {
	a := newTestApp(runtime.GenerationResult{Text: "Xin chào"}, nil)
	var out, errb bytes.Buffer
	code := Run(a, []string{"Hello"}, nil, &out, &errb)
	if code != 0 {
		t.Fatalf("code %d err %s", code, errb.String())
	}
	if strings.TrimSpace(out.String()) != "Xin chào" {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestCLIStderrOnError(t *testing.T) {
	a := newTestApp(runtime.GenerationResult{}, &runtime.RuntimeError{Kind: runtime.ErrProviderUnavail, Err: io.EOF})
	var out, errb bytes.Buffer
	code := Run(a, []string{"Hello"}, nil, &out, &errb)
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	if errb.Len() == 0 {
		t.Fatal("expected stderr")
	}
}

func TestCLIStdin(t *testing.T) {
	a := newTestApp(runtime.GenerationResult{Text: "hi"}, nil)
	var out, errb bytes.Buffer
	code := Run(a, []string{}, strings.NewReader("Hello\n"), &out, &errb)
	if code != 0 {
		t.Fatalf("code %d", code)
	}
	if strings.TrimSpace(out.String()) != "hi" {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestCLINoInputReturnsMinus1(t *testing.T) {
	a := newTestApp(runtime.GenerationResult{Text: "hi"}, nil)
	var out, errb bytes.Buffer
	code := Run(a, []string{}, nil, &out, &errb)
	if code != -1 {
		t.Fatalf("code %d", code)
	}
}

func TestCLIHelpExitsZero(t *testing.T) {
	a := newTestApp(runtime.GenerationResult{}, nil)
	var out, errb bytes.Buffer
	code := Run(a, []string{"--help"}, nil, &out, &errb)
	if code != 0 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(errb.String(), "Usage of tnow") {
		t.Fatalf("stderr=%q", errb.String())
	}
}

func TestCLICountryNameLanguage(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TNOW_CONFIG", filepath.Join(dir, "config.yaml"))
	t.Setenv("TNOW_JSON_CONFIG", filepath.Join(dir, "config.json"))

	a := newTestApp(runtime.GenerationResult{Text: "Xin chào"}, nil)
	var out, errb bytes.Buffer
	code := Run(a, []string{"--to", "vietnam", "Hello"}, nil, &out, &errb)
	if code != 0 {
		t.Fatalf("code %d err %s", code, errb.String())
	}
	if a.TargetLang != language.VI {
		t.Fatalf("target = %q, want Vietnamese", a.TargetLang)
	}
}

func TestCLIWordMode(t *testing.T) {
	resObj := analysis.Result{Word: "awkward", Language: "en", IPA: "/x/", PartOfSpeech: "adjective", Definitions: []analysis.Definition{{Meaning: "m", Translations: []string{"v"}}}, Examples: []analysis.Example{{Source: "s", Translation: "t"}}}
	b, _ := json.Marshal(resObj)
	a := newTestApp(runtime.GenerationResult{Text: string(b)}, nil)
	var out, errb bytes.Buffer
	code := Run(a, []string{"--word", "awkward"}, nil, &out, &errb)
	if code != 0 {
		t.Fatalf("code %d err %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "awkward") {
		t.Fatalf("stdout=%q", out.String())
	}
}
