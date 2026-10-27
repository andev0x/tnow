package translation

import (
	"context"
	"strings"

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
	return &Engine{Runtime: rt, Provider: provider, Model: model, MaxTokens: 2048, Temperature: 0.2}
}

func (e *Engine) Translate(ctx context.Context, req Request) (Result, error) {
	// resolve source language if auto
	src := req.SourceLang
	if src == "" {
		src = language.Auto
	}
	if src == language.Auto {
		det, _ := language.Detect(req.Text)
		if det != language.Auto {
			src = det
		}
	}
	gr := BuildRequest(req, e.Provider, e.Model, e.MaxTokens, e.Temperature, false)
	res, err := e.Runtime.Execute(ctx, gr)
	if err != nil {
		return Result{}, err
	}
	out, err := Validate(res.Text)
	if err != nil {
		return Result{}, &runtime.RuntimeError{Kind: runtime.ErrValidation, Err: err}
	}
	return Result{Text: out, SourceLang: src, TargetLang: req.TargetLang}, nil
}

func (e *Engine) TranslateStream(ctx context.Context, req Request) (<-chan string, <-chan error, language.Language, language.Language, error) {
	src := req.SourceLang
	if src == "" {
		src = language.Auto
	}
	if src == language.Auto {
		det, _ := language.Detect(req.Text)
		if det != language.Auto {
			src = det
		}
	}
	gr := BuildRequest(req, e.Provider, e.Model, e.MaxTokens, e.Temperature, true)
	res, err := e.Runtime.Execute(ctx, gr)
	if err != nil {
		return nil, nil, src, req.TargetLang, err
	}
	outCh := make(chan string, 32)
	errCh := make(chan error, 1)
	go func() {
		defer close(outCh)
		defer close(errCh)
		var sb strings.Builder
		_ = sb
		if res.Stream == nil {
			// fallback
			o, verr := Validate(res.Text)
			if verr != nil {
				errCh <- verr
				return
			}
			outCh <- o
			return
		}
		for ev := range res.Stream {
			if ev.Err != nil {
				errCh <- ev.Err
				return
			}
			if ev.Chunk != "" {
				outCh <- ev.Chunk
			}
			if ev.Done {
				return
			}
		}
	}()
	return outCh, errCh, src, req.TargetLang, nil
}
