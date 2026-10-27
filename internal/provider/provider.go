package provider

import (
	"time"

	"github.com/andev0x/tnow/internal/runtime"
)

// Config holds provider connection settings.
type Config struct {
	Name    string
	BaseURL string
	APIKey  string
	Model   string
	Org     string
	Project string
	Timeout time.Duration
}

// New creates a provider by name.
func New(cfg Config) (runtime.Provider, error) {
	switch cfg.Name {
	case "ollama":
		return NewOllama(cfg), nil
	case "openai", "openai-compatible", "groq", "gemini", "openrouter":
		return NewOpenAICompatible(cfg), nil
	default:
		return nil, &runtime.RuntimeError{Kind: runtime.ErrConfig, Err: errUnknownProvider(cfg.Name)}
	}
}

type simpleErr string

func (s simpleErr) Error() string { return string(s) }

func errUnknownProvider(name string) error {
	return simpleErr("unknown provider: " + name)
}
