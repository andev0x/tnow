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
		target = language.VI
	}
	src := req.SourceLang
	if src == "" {
		src = language.Auto
	}
	if src == language.Auto {
		det, _ := language.Detect(req.Word)
		if det != language.Auto {
			src = det
		} else {
			src = language.EN
		}
	}
	gr := runtime.GenerationRequest{
		Provider:     e.Provider,
		Model:        e.Model,
		SystemPrompt: JSONSchema(target),
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
	if parsed.Language == "" {
		parsed.Language = string(src)
	}
	if err := Validate(parsed); err != nil {
		return Result{}, &runtime.RuntimeError{Kind: runtime.ErrValidation, Err: err}
	}
	return parsed, nil
}
