package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.DefaultProvider != "ollama" {
		t.Fatal("bad default")
	}
}

func TestLoadMissing(t *testing.T) {
	os.Setenv("TNOW_CONFIG", "/tmp/tnow-nonexistent.yaml")
	defer os.Unsetenv("TNOW_CONFIG")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.DefaultProvider != "ollama" {
		t.Fatal("bad")
	}
}

func TestSaveWritesJSONMirrorWithPrivatePermissions(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "config.json")
	t.Setenv("TNOW_CONFIG", filepath.Join(dir, "config.yaml"))
	t.Setenv("TNOW_JSON_CONFIG", jsonPath)

	cfg := Default()
	cfg.SetAPIKey("openai", "secret-key")
	if err := Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read JSON mirror: %v", err)
	}
	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("parse JSON mirror: %v", err)
	}
	provider, ok := saved.Provider("openai")
	if !ok || provider.APIKey != "secret-key" {
		t.Fatalf("JSON mirror did not contain the API key: %+v", provider)
	}
	info, err := os.Stat(jsonPath)
	if err != nil {
		t.Fatalf("stat JSON mirror: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("JSON config permissions = %o, want 600", got)
	}
}

func TestEnvOverride(t *testing.T) {
	os.Setenv("GROQ_API_KEY", "secret")
	defer os.Unsetenv("GROQ_API_KEY")
	os.Setenv("TNOW_CONFIG", "/tmp/tnow-nonexistent2.yaml")
	defer os.Unsetenv("TNOW_CONFIG")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	pc, ok := c.Provider("groq")
	if !ok || pc.APIKey != "secret" {
		t.Fatalf("groq not configured: %+v", pc)
	}
}
