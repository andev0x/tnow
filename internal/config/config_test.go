package config

import (
	"os"
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
