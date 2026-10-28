package analysis

import (
	"context"
	"fmt"

	"github.com/andev0x/tnow/internal/language"
	"github.com/andev0x/tnow/internal/runtime"
)

type Engine struct {
	Runtime     *runtime.Runtime
	Provider    string
	Model       string
	MaxTokens   int
	Temperature float64
}

func NewEngine(rt *runtime.Runtime, provider, model string) *Engine {
	return &Engine{Runtime: rt, Provider: provider, Model: model, MaxTokens: 1024, Temperature: 0.2}
}

func (e *Engine) Analyze(ctx context.Context, req Request) (Result, error) {
	target := req.TargetLang
	if target == "" {
		target = language.EN
	}
	src := req.SourceLang
	if src == "" {
		src = language.Auto
	}
	if src == language.Auto {
		// Leave ambiguous Latin-script input as auto so the model can identify
		// it; a detected script or Vietnamese diacritics are reliable enough to
		// name in the prompt (and to trust over the model's own label).
		det, _ := language.Detect(req.Word)
		if det != language.Auto && det != language.EN {
			src = det
		}
	}
	gr := runtime.GenerationRequest{
		Provider:     e.Provider,
		Model:        e.Model,
		SystemPrompt: JSONSchema(src, target),
		UserPrompt:   BuildUserPrompt(req.Word),
		MaxTokens:    e.MaxTokens,
		Temperature:  e.Temperature,
		Stream:       false,
		Operation:    "analysis",
	}
	res, err := e.Runtime.Execute(ctx, gr)
	if err != nil {
		return Result{}, err
	}
	parsed, err := Parse(res.Text)
	if err != nil {
		return Result{}, &runtime.RuntimeError{Kind: runtime.ErrValidation, Err: fmt.Errorf("parse analysis: %w", err)}
	}
	if parsed.Word == "" {
		parsed.Word = req.Word
	}
	// The resolved source language is authoritative. Models sometimes answer
	// with a different language label (for example Chinese for Vietnamese),
	// which would contradict the request metadata shown in the UI.
	if src != language.Auto {
		parsed.Language = language.DisplayName(src)
	}
	if err := Validate(parsed); err != nil {
		return Result{}, &runtime.RuntimeError{Kind: runtime.ErrValidation, Err: err}
	}
	return parsed, nil
}
