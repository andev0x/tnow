package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
