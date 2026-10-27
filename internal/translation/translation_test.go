package translation

import (
	"context"
	"testing"

	"github.com/andev0x/tnow/internal/language"
	"github.com/andev0x/tnow/internal/runtime"
)

type mockProvider struct {
	res runtime.GenerationResult
	err error
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Generate(ctx context.Context, req runtime.GenerationRequest) (runtime.GenerationResult, error) {
	return m.res, m.err
}

func TestValidateEmpty(t *testing.T) {
	if _, err := Validate(""); err == nil {
		t.Fatal("expected err")
	}
}

func TestValidateFiller(t *testing.T) {
	if _, err := Validate("Here is the translation: hi"); err == nil {
		t.Fatal("expected err")
	}
}

func TestValidateOK(t *testing.T) {
	out, err := Validate("  hello  \n")
	if err != nil || out != "hello" {
		t.Fatalf("got %q %v", out, err)
	}
}

func TestNormalizeMultiline(t *testing.T) {
	in := "a\nb\nc"
	out := Normalize(in)
	if out != "a\nb\nc" {
		t.Fatalf("got %q", out)
	}
}

func TestEngineTranslate(t *testing.T) {
	mp := &mockProvider{res: runtime.GenerationResult{Text: "Xin chào"}}
	rt := runtime.New(map[string]runtime.Provider{"mock": mp}, "mock")
	eng := NewEngine(rt, "mock", "m")
	r, err := eng.Translate(context.Background(), Request{Text: "Hello", TargetLang: language.VI})
	if err != nil || r.Text != "Xin chào" {
		t.Fatalf("got %q %v", r.Text, err)
	}
}

func TestEngineTranslateEmptyOutput(t *testing.T) {
	mp := &mockProvider{res: runtime.GenerationResult{Text: "   "}}
	rt := runtime.New(map[string]runtime.Provider{"mock": mp}, "mock")
	eng := NewEngine(rt, "mock", "m")
	_, err := eng.Translate(context.Background(), Request{Text: "Hello", TargetLang: language.VI})
	if err == nil {
		t.Fatal("expected err")
	}
}

func TestSystemPrompt(t *testing.T) {
	p := SystemPrompt(language.VI)
	if !contains(p, "Vietnamese") {
		t.Fatalf("missing target: %s", p)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
