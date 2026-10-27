package runtime

import (
	"context"
	"fmt"
)

// GenerationRequest is the bounded contract for a single model generation.
type GenerationRequest struct {
	Provider string
	Model    string

	SystemPrompt string
	UserPrompt   string

	MaxTokens   int
	Temperature float64

	Stream bool

	// Operation labels the kind of work (translation, analysis, ...).
	Operation string
}

// GenerationResult is the outcome of a generation.
type GenerationResult struct {
	Text   string
	Usage  Usage
	Stream <-chan StreamEvent
	Cancel context.CancelFunc
}

// Usage holds token counts when available.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// StreamEvent carries a chunk or terminal status from a streaming generation.
type StreamEvent struct {
	Chunk string
	Err   error
	Done  bool
	Usage Usage
}

// Provider is the inference transport boundary.
type Provider interface {
	Name() string
	Generate(ctx context.Context, req GenerationRequest) (GenerationResult, error)
}

// ModelLister is implemented by providers that can enumerate their models.
type ModelLister interface {
	ListModels(ctx context.Context) ([]string, error)
}

// Runtime executes bounded generation requests.
type Runtime struct {
	Providers map[string]Provider
	Default   string
	Timeout   int // seconds
}

func New(providers map[string]Provider, defaultName string) *Runtime {
	if providers == nil {
		providers = map[string]Provider{}
	}
	return &Runtime{Providers: providers, Default: defaultName, Timeout: 60}
}

// ListModels asks a provider for its available models. Providers that do not
// support model discovery return a configuration error.
func (r *Runtime) ListModels(ctx context.Context, providerName string) ([]string, error) {
	if providerName == "" {
		providerName = r.Default
	}
	p, ok := r.Providers[providerName]
	if !ok {
		return nil, &RuntimeError{Kind: ErrConfig, Err: fmt.Errorf("unknown provider %q", providerName)}
	}
	lister, ok := p.(ModelLister)
	if !ok {
		return nil, &RuntimeError{Kind: ErrConfig, Err: fmt.Errorf("provider %q does not support model listing", providerName)}
	}
	return lister.ListModels(ctx)
}
