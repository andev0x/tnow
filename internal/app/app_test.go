package app

import (
	"path/filepath"
	"testing"

	"github.com/andev0x/tnow/internal/config"
)

func isolateConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TNOW_CONFIG", filepath.Join(dir, "config.yaml"))
	t.Setenv("TNOW_JSON_CONFIG", filepath.Join(dir, "config.json"))
}

func TestSetProviderDoesNotCarryModelAcrossProviders(t *testing.T) {
	isolateConfig(t)

	cfg := config.Config{
		DefaultProvider: "ollama",
		DefaultModel:    "translategemma:4b",
		Providers: []config.ProviderConfig{
			{Name: "ollama", BaseURL: "http://localhost:11434", Model: "translategemma:4b"},
			{Name: "gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai"},
		},
	}
	a, err := New(cfg, "", "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	a.SetProvider("gemini")
	if a.Model != "" {
		t.Fatalf("model = %q, want empty after switching to a provider without a model", a.Model)
	}
	if a.Translation.Model != "" || a.Analysis.Model != "" {
		t.Fatalf("engines kept the previous model: %q / %q", a.Translation.Model, a.Analysis.Model)
	}
}

func TestSetProviderRestoresProviderModel(t *testing.T) {
	isolateConfig(t)

	cfg := config.Config{
		DefaultProvider: "ollama",
		DefaultModel:    "translategemma:4b",
		Providers: []config.ProviderConfig{
			{Name: "ollama", BaseURL: "http://localhost:11434", Model: "translategemma:4b"},
			{Name: "gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", Model: "gemini-3.5-flash-lite"},
		},
	}
	a, err := New(cfg, "", "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	a.SetProvider("gemini")
	if a.Model != "gemini-3.5-flash-lite" {
		t.Fatalf("model = %q, want gemini-3.5-flash-lite", a.Model)
	}
}
