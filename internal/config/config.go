package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProviderConfig struct {
	Name    string `yaml:"name" json:"name"`
	BaseURL string `yaml:"base_url" json:"base_url"`
	APIKey  string `yaml:"api_key" json:"api_key"`
	Model   string `yaml:"model" json:"model"`
	Org     string `yaml:"org" json:"org"`
	Project string `yaml:"project" json:"project"`
	Timeout int    `yaml:"timeout" json:"timeout"`
}

type Config struct {
	DefaultProvider string            `yaml:"default_provider" json:"default_provider"`
	DefaultModel    string            `yaml:"default_model" json:"default_model"`
	SourceLang      string            `yaml:"source_lang" json:"source_lang"`
	TargetLang      string            `yaml:"target_lang" json:"target_lang"`
	Theme           string            `yaml:"theme" json:"theme"`
	ColorTheme      string            `yaml:"color_theme" json:"color_theme"`
	ThemeName       string            `yaml:"theme_name" json:"theme_name"`
	ColorScheme     string            `yaml:"color_scheme" json:"color_scheme"`
	Colors          map[string]string `yaml:"colors" json:"colors"`
	Providers       []ProviderConfig  `yaml:"providers" json:"providers"`
	// runtime
	Timeout int `yaml:"timeout" json:"timeout"`
	// auto copy translation output to clipboard when done
	AutoCopy bool `yaml:"auto_copy" json:"auto_copy"`
}

func Default() Config {
	return Config{
		DefaultProvider: "ollama",
		DefaultModel:    "translategemma:4b",
		SourceLang:      "auto",
		TargetLang:      "en",
		Theme:           "tokyo-night",
		Timeout:         60,
		AutoCopy:        true,
		Providers: []ProviderConfig{
			{Name: "ollama", BaseURL: "http://localhost:11434", Model: "translategemma:4b"},
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

func JSONPath() string {
	if p := os.Getenv("TNOW_JSON_CONFIG"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".tnow", "config.json")
}

// jsonConfigEnabled reports whether the JSON mirror should participate in
// loading/saving. Tests can isolate the YAML path with TNOW_CONFIG while still
// opting into a temporary JSON path through TNOW_JSON_CONFIG.
func jsonConfigEnabled() bool {
	return os.Getenv("TNOW_CONFIG") == "" || os.Getenv("TNOW_JSON_CONFIG") != ""
}

func Load() (Config, error) {
	cfg := Default()
	// Try primary YAML path first
	p := Path()
	jp := JSONPath()
	yamlLoaded := false
	yamlHasAutoCopy := false
	if p != "" {
		if data, err := os.ReadFile(p); err == nil {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return cfg, fmt.Errorf("parse config: %w", err)
			}
			yamlLoaded = true
			// detect if auto_copy was explicitly set; otherwise default to true
			if strings.Contains(string(data), "auto_copy") {
				yamlHasAutoCopy = true
			}
		} else if !os.IsNotExist(err) {
			return cfg, err
		}
	}
	// Skip JSON fallback when TNOW_CONFIG is set for test isolation
	useJSON := jp != "" && jsonConfigEnabled()
	// Sync from JSON if YAML not found, or merge JSON as fallback
	if !yamlLoaded && useJSON {
		if data, err := os.ReadFile(jp); err == nil {
			// JSON uses same struct; try unmarshal
			var jcfg Config
			if err := json.Unmarshal(data, &jcfg); err == nil {
				// default auto_copy true if not present
				if !strings.Contains(string(data), "auto_copy") {
					jcfg.AutoCopy = true
				}
				// merge json config as base if yaml missing
				cfg = jcfg
				if cfg.DefaultProvider == "" {
					cfg.DefaultProvider = "ollama"
				}
				if cfg.Theme == "" {
					cfg.Theme = "tokyo-night"
				}
				if cfg.Timeout <= 0 {
					cfg.Timeout = 60
				}
				// sync back to YAML path for consistency
				_ = Save(cfg)
				applyEnv(&cfg)
				return cfg, nil
			}
		}
	}
	// If YAML was loaded, also check JSON for newer data and merge (JSON wins for providers if present)
	// Skip JSON sync when TNOW_CONFIG is explicitly set for test isolation unless TNOW_JSON_CONFIG also set
	if yamlLoaded && jp != "" && jsonConfigEnabled() {
		if data, err := os.ReadFile(jp); err == nil {
			var jcfg Config
			if err := json.Unmarshal(data, &jcfg); err == nil && len(jcfg.Providers) > 0 {
				// simple sync: if JSON has providers not in YAML, merge them
				// For full sync, if JSON is newer, prefer its values for overlapping keys
				// Here we merge providers by name, preferring JSON's APIKey/Model/BaseURL if non-empty
				merged := cfg
				for _, jp := range jcfg.Providers {
					found := false
					for i, cp := range merged.Providers {
						if cp.Name == jp.Name {
							if jp.APIKey != "" {
								merged.Providers[i].APIKey = jp.APIKey
							}
							if jp.Model != "" {
								merged.Providers[i].Model = jp.Model
							}
							if jp.BaseURL != "" {
								merged.Providers[i].BaseURL = jp.BaseURL
							}
							found = true
							break
						}
					}
					if !found {
						merged.Providers = append(merged.Providers, jp)
					}
				}
				if jcfg.DefaultProvider != "" {
					merged.DefaultProvider = jcfg.DefaultProvider
				}
				if jcfg.DefaultModel != "" {
					merged.DefaultModel = jcfg.DefaultModel
				}
				if jcfg.Theme != "" {
					merged.Theme = jcfg.Theme
				}
				if len(jcfg.Colors) > 0 {
					if merged.Colors == nil {
						merged.Colors = map[string]string{}
					}
					for k, v := range jcfg.Colors {
						merged.Colors[k] = v
					}
				}
				cfg = merged
			}
		}
	}
	// merge defaults for missing
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = "ollama"
	}
	if cfg.Theme == "" {
		cfg.Theme = "tokyo-night"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60
	}
	if yamlLoaded && !yamlHasAutoCopy {
		cfg.AutoCopy = true
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

func Save(cfg Config) error {
	// Save to primary YAML path
	p := Path()
	if p == "" {
		return fmt.Errorf("no config path")
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(p, 0o600); err != nil {
		return err
	}
	// Sync to JSON path ~/.tnow/config.json (skip when TNOW_CONFIG is set for test isolation)
	if jsonConfigEnabled() {
		jp := JSONPath()
		if jp != "" {
			jdir := filepath.Dir(jp)
			if err := os.MkdirAll(jdir, 0o700); err == nil {
				jdata, err := json.MarshalIndent(cfg, "", "  ")
				if err == nil {
					_ = os.WriteFile(jp, jdata, 0o600)
					_ = os.Chmod(jp, 0o600)
				}
			}
		}
	}
	return nil
}

func SaveJSON(cfg Config) error {
	jp := JSONPath()
	if jp == "" {
		return fmt.Errorf("no json config path")
	}
	dir := filepath.Dir(jp)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jp, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(jp, 0o600)
}

func LoadJSON() (Config, error) {
	jp := JSONPath()
	if jp == "" {
		return Default(), fmt.Errorf("no json config path")
	}
	cfg := Default()
	data, err := os.ReadFile(jp)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse json config: %w", err)
	}
	return cfg, nil
}

func (c *Config) SetAPIKey(name, key string) {
	for i := range c.Providers {
		if c.Providers[i].Name == name {
			c.Providers[i].APIKey = key
			return
		}
	}
	pc := ProviderConfig{Name: name, APIKey: key}
	switch name {
	case "groq":
		pc.BaseURL = "https://api.groq.com/openai/v1"
	case "gemini":
		pc.BaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	case "openrouter":
		pc.BaseURL = "https://openrouter.ai/api/v1"
	case "openai":
		pc.BaseURL = "https://api.openai.com/v1"
	}
	c.Providers = append(c.Providers, pc)
}

func ProviderRequiresKey(name string) bool {
	switch name {
	case "ollama", "":
		return false
	default:
		return true
	}
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
