package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

// errorMessage extracts the human-readable message from a provider error body.
// OpenAI-compatible endpoints return {"error":{"message":...}} while some
// providers (Gemini, for example) wrap the same object in a JSON array. When no
// message can be decoded the trimmed body is returned so callers still see
// something useful.
func errorMessage(body []byte) string {
	type apiError struct {
		Message string `json:"message"`
	}
	one := struct {
		Error apiError `json:"error"`
	}{}
	many := []struct {
		Error apiError `json:"error"`
	}{}
	switch {
	case json.Unmarshal(body, &one) == nil && one.Error.Message != "":
		return one.Error.Message
	case json.Unmarshal(body, &many) == nil && len(many) > 0 && many[0].Error.Message != "":
		return many[0].Error.Message
	}
	return strings.TrimSpace(string(body))
}

// statusError classifies a non-2xx provider response and keeps only the
// provider's own message instead of dumping the raw response body.
func statusError(code int, body []byte, model string) *runtime.RuntimeError {
	kind := runtime.ErrModel
	switch {
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		kind = runtime.ErrAuth
	case code == http.StatusTooManyRequests:
		kind = runtime.ErrRateLimit
	case code >= 500:
		kind = runtime.ErrProviderUnavail
	}
	msg := errorMessage(body)
	if msg == "" {
		msg = http.StatusText(code)
	}
	if code == http.StatusNotFound && model != "" {
		msg = fmt.Sprintf("%s (model %q is not available; choose another model)", msg, model)
	}
	return &runtime.RuntimeError{Kind: kind, Err: fmt.Errorf("status %d: %s", code, msg)}
}

type simpleErr string

func (s simpleErr) Error() string { return string(s) }

func errUnknownProvider(name string) error {
	return simpleErr("unknown provider: " + name)
}
