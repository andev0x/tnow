package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProviderConfig struct {
	Name    string `yaml:"name"`
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	Org     string `yaml:"org"`
	Project string `yaml:"project"`
	Timeout int    `yaml:"timeout"`
}

type Config struct {
	DefaultProvider string           `yaml:"default_provider"`
	DefaultModel    string           `yaml:"default_model"`
	SourceLang      string           `yaml:"source_lang"`
	TargetLang      string           `yaml:"target_lang"`
	Providers       []ProviderConfig `yaml:"providers"`
	// runtime
	Timeout int `yaml:"timeout"`
}

func Default() Config {
	return Config{
		DefaultProvider: "ollama",
		DefaultModel:    "",
		SourceLang:      "auto",
		TargetLang:      "vi",
		Timeout:         60,
		Providers: []ProviderConfig{
			{Name: "ollama", BaseURL: "http://localhost:11434", Model: "llama3.2"},
		},
	}
}

func Path() string {
	if p := os.Getenv("TNOW_CONFIG"); p != "" {
		return p
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "tnow", "config.yaml")
}

func Load() (Config, error) {
	cfg := Default()
	p := Path()
	if p == "" {
		applyEnv(&cfg)
		return cfg, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnv(&cfg)
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	// merge defaults for missing
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = "ollama"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60
	}
	applyEnv(&cfg)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("OLLAMA_HOST"); v != "" {
		for i := range cfg.Providers {
			if cfg.Providers[i].Name == "ollama" {
				cfg.Providers[i].BaseURL = v
			}
		}
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		ensureProvider(cfg, "openai")
		setKey(cfg, "openai", v)
	}
	if v := os.Getenv("GROQ_API_KEY"); v != "" {
		ensureProvider(cfg, "groq")
		setKey(cfg, "groq", v)
	}
	if v := os.Getenv("GEMINI_API_KEY"); v != "" {
		ensureProvider(cfg, "gemini")
		setKey(cfg, "gemini", v)
	}
	if v := os.Getenv("OPENROUTER_API_KEY"); v != "" {
		ensureProvider(cfg, "openrouter")
		setKey(cfg, "openrouter", v)
	}
	// base urls for known
	for i := range cfg.Providers {
		switch cfg.Providers[i].Name {
		case "groq":
			if cfg.Providers[i].BaseURL == "" {
				cfg.Providers[i].BaseURL = "https://api.groq.com/openai/v1"
			}
		case "gemini":
			if cfg.Providers[i].BaseURL == "" {
				cfg.Providers[i].BaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
			}
		case "openrouter":
			if cfg.Providers[i].BaseURL == "" {
				cfg.Providers[i].BaseURL = "https://openrouter.ai/api/v1"
			}
		case "openai":
			if cfg.Providers[i].BaseURL == "" {
				cfg.Providers[i].BaseURL = "https://api.openai.com/v1"
			}
		}
	}
}

func ensureProvider(cfg *Config, name string) {
	for _, p := range cfg.Providers {
		if p.Name == name {
			return
		}
	}
	cfg.Providers = append(cfg.Providers, ProviderConfig{Name: name})
}

func setKey(cfg *Config, name, key string) {
	for i := range cfg.Providers {
		if cfg.Providers[i].Name == name {
			if cfg.Providers[i].APIKey == "" {
				cfg.Providers[i].APIKey = key
			}
		}
	}
}

func (c Config) Provider(name string) (ProviderConfig, bool) {
	for _, p := range c.Providers {
		if p.Name == name {
			return p, true
		}
	}
	return ProviderConfig{}, false
}

// CLI override merge helper.
func MergeProvider(cfg Config, name, model, baseURL, apiKey string) Config {
	if name != "" {
		cfg.DefaultProvider = name
	}
	if pc, ok := cfg.Provider(cfg.DefaultProvider); ok {
		if model != "" {
			pc.Model = model
		}
		if baseURL != "" {
			pc.BaseURL = baseURL
		}
		if apiKey != "" {
			pc.APIKey = apiKey
		}
		// write back
		for i := range cfg.Providers {
			if cfg.Providers[i].Name == cfg.DefaultProvider {
				cfg.Providers[i] = pc
			}
		}
		if _, ok := cfg.Provider(cfg.DefaultProvider); !ok {
			cfg.Providers = append(cfg.Providers, pc)
		}
	} else if name != "" {
		cfg.Providers = append(cfg.Providers, ProviderConfig{Name: name, BaseURL: baseURL, APIKey: apiKey, Model: model})
	}
	if model != "" && cfg.DefaultModel == "" {
		cfg.DefaultModel = model
	}
	_ = strings.TrimSpace
	return cfg
}
