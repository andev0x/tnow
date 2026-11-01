package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andev0x/tnow/internal/runtime"
)

func TestOpenAICompatibleGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "bonjour"}}},
			"usage":   map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	}))
	defer srv.Close()

	p := NewOpenAICompatible(Config{Name: "openai", BaseURL: srv.URL, APIKey: "k", Model: "m"})
	res, err := p.Generate(context.Background(), runtime.GenerationRequest{Model: "m", UserPrompt: "hi", Temperature: 0.2})
	if err != nil || res.Text != "bonjour" {
		t.Fatalf("got %q %v", res.Text, err)
	}
}

func TestOllamaGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message":           map[string]string{"content": "xin chao"},
			"done":              true,
			"prompt_eval_count": 1,
			"eval_count":        2,
		})
	}))
	defer srv.Close()

	p := NewOllama(Config{Name: "ollama", BaseURL: srv.URL, Model: "m"})
	res, err := p.Generate(context.Background(), runtime.GenerationRequest{Model: "m", UserPrompt: "hi", Temperature: 0.2})
	if err != nil || res.Text != "xin chao" {
		t.Fatalf("got %q %v", res.Text, err)
	}
}

func TestStatusErrorExtractsProviderMessage(t *testing.T) {
	tests := []struct {
		name string
		code int
		body string
		want string
		kind runtime.ErrorKind
	}{
		{
			name: "openai object",
			code: http.StatusBadRequest,
			body: `{"error":{"message":"bad request","type":"invalid_request_error"}}`,
			want: "status 400: bad request",
			kind: runtime.ErrModel,
		},
		{
			name: "gemini array",
			code: http.StatusBadRequest,
			body: `[{"error":{"code":400,"message":"model is retired"}}]`,
			want: "status 400: model is retired",
			kind: runtime.ErrModel,
		},
		{
			name: "unauthorized",
			code: http.StatusUnauthorized,
			body: `{"error":{"message":"invalid key"}}`,
			want: "status 401: invalid key",
			kind: runtime.ErrAuth,
		},
		{
			name: "fallback to raw body",
			code: http.StatusBadGateway,
			body: "upstream exploded",
			want: "status 502: upstream exploded",
			kind: runtime.ErrProviderUnavail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := statusError(tt.code, []byte(tt.body), "")
			if err.Kind != tt.kind {
				t.Fatalf("kind = %q, want %q", err.Kind, tt.kind)
			}
			if got := err.Err.Error(); got != tt.want {
				t.Fatalf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusErrorNotFoundMentionsModel(t *testing.T) {
	err := statusError(http.StatusNotFound, []byte(`[{"error":{"message":"no longer available"}}]`), "models/retired")
	if !strings.Contains(err.Error(), "no longer available") || !strings.Contains(err.Error(), `model "models/retired" is not available`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), `"code"`) {
		t.Fatalf("raw JSON leaked into message: %v", err)
	}
}

func TestOpenAICompatibleGenerateModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`[{"error":{"code":404,"message":"This model models/retired is no longer available to new users."}}]`))
	}))
	defer srv.Close()

	p := NewOpenAICompatible(Config{Name: "gemini", BaseURL: srv.URL, APIKey: "k", Model: "models/retired"})
	_, err := p.Generate(context.Background(), runtime.GenerationRequest{Model: "models/retired", UserPrompt: "hi"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "no longer available") {
		t.Fatalf("provider message missing from error: %v", err)
	}
}
