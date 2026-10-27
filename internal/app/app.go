package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/andev0x/tnow/internal/analysis"
	"github.com/andev0x/tnow/internal/config"
	"github.com/andev0x/tnow/internal/language"
	"github.com/andev0x/tnow/internal/provider"
	"github.com/andev0x/tnow/internal/runtime"
	"github.com/andev0x/tnow/internal/translation"
)

type App struct {
	Config config.Config

	Runtime *runtime.Runtime

	Translation *translation.Engine
	Analysis    *analysis.Engine

	SourceLang language.Language
	TargetLang language.Language
	Provider   string
	Model      string
}

func New(cfg config.Config, providerOverride, modelOverride string) (*App, error) {
	cfg = config.MergeProvider(cfg, providerOverride, modelOverride, "", "")
	// build providers map
	providers := map[string]runtime.Provider{}
	for _, pc := range cfg.Providers {
		pcfg := provider.Config{
			Name:    pc.Name,
			BaseURL: pc.BaseURL,
			APIKey:  pc.APIKey,
			Model:   pc.Model,
			Org:     pc.Org,
			Project: pc.Project,
			Timeout: timeDuration(pc.Timeout),
		}
		p, err := provider.New(pcfg)
		if err != nil {
			continue
		}
		providers[pc.Name] = p
	}
	rt := runtime.New(providers, cfg.DefaultProvider)
	rt.Timeout = cfg.Timeout

	a := &App{
		Config:     cfg,
		Runtime:    rt,
		SourceLang: language.Normalize(cfg.SourceLang),
		TargetLang: language.Normalize(cfg.TargetLang),
		Provider:   cfg.DefaultProvider,
		Model:      cfg.DefaultModel,
	}
	// resolve model from provider if empty
	if a.Model == "" {
		if pc, ok := cfg.Provider(a.Provider); ok {
			a.Model = pc.Model
		}
	}
	a.Translation = translation.NewEngine(rt, a.Provider, a.Model)
	a.Analysis = analysis.NewEngine(rt, a.Provider, a.Model)
	return a, nil
}

func (a *App) SetProvider(name string) {
	a.Provider = name
	a.Config.DefaultProvider = name
	// ensure provider entry exists so it can persist
	if _, ok := a.Config.Provider(name); !ok {
		pc := config.ProviderConfig{Name: name}
		switch name {
		case "groq":
			pc.BaseURL = "https://api.groq.com/openai/v1"
		case "gemini":
			pc.BaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
		case "openrouter":
			pc.BaseURL = "https://openrouter.ai/api/v1"
		case "openai":
			pc.BaseURL = "https://api.openai.com/v1"
		case "ollama":
			pc.BaseURL = "http://localhost:11434"
		}
		a.Config.Providers = append(a.Config.Providers, pc)
	}
	// restore the most recently selected model for this provider so it
	// doesn't change when returning later
	if pc, ok := a.Config.Provider(name); ok && pc.Model != "" {
		a.Model = pc.Model
	}
	a.Translation.Provider = a.Provider
	a.Translation.Model = a.Model
	a.Analysis.Provider = a.Provider
	a.Analysis.Model = a.Model
	_ = config.Save(a.Config)
}

func (a *App) SetModel(model string) {
	a.Model = model
	a.Translation.Model = model
	a.Analysis.Model = model
	// persist: update DefaultModel and provider-specific model
	a.Config.DefaultModel = model
	updated := false
	for i, pc := range a.Config.Providers {
		if pc.Name == a.Provider {
			a.Config.Providers[i].Model = model
			updated = true
			break
		}
	}
	if !updated {
		a.Config.Providers = append(a.Config.Providers, config.ProviderConfig{Name: a.Provider, Model: model})
	}
	_ = config.Save(a.Config)
}

func (a *App) SetTarget(lang language.Language) {
	a.TargetLang = lang
}

func (a *App) SetSource(lang language.Language) {
	a.SourceLang = lang
}

// ProviderNames returns the configured providers in a stable order.
func (a *App) ProviderNames() []string {
	seen := make(map[string]bool)
	var names []string
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	for _, pc := range a.Config.Providers {
		add(pc.Name)
	}
	for name := range a.Runtime.Providers {
		add(name)
	}
	add(a.Provider)
	sort.Strings(names)
	return names
}

// ModelsForProvider returns locally configured models for a provider. The
// active model is included even when it is not present in the config file.
func (a *App) ModelsForProvider(name string) []string {
	seen := make(map[string]bool)
	var models []string
	add := func(model string) {
		if model == "" || seen[model] {
			return
		}
		seen[model] = true
		models = append(models, model)
	}
	for _, pc := range a.Config.Providers {
		if pc.Name == name {
			add(pc.Model)
		}
	}
	if name == a.Provider {
		add(a.Model)
	}
	sort.Strings(models)
	return models
}

// ListModels asks the active provider for the models it exposes.
func (a *App) ListModels(ctx context.Context) ([]string, error) {
	return a.Runtime.ListModels(ctx, a.Provider)
}

// IsTranslationExcellent reports whether a model is known to excel at multilingual translation.
func IsTranslationExcellent(model string) bool {
	m := strings.ToLower(model)
	markers := []string{
		"gpt-4o", "gpt-4", "claude", "gemini", "qwen", "nllb", "aya",
		"mistral", "command-r", "llama3", "llama-3", "seamless", "opus-mt",
		"m2m", "nmt", "deepl", "phi-3", "cohere",
	}
	for _, mk := range markers {
		if strings.Contains(m, mk) {
			return true
		}
	}
	return false
}

// ModelsForProviderFiltered returns models sorted with translation-excellent first.
func (a *App) ModelsForProviderFiltered(name string, onlyExcellent bool) []string {
	all := a.ModelsForProvider(name)
	if !onlyExcellent {
		// sort excellent first, then rest, both alphabetically
		var excellent, other []string
		for _, m := range all {
			if IsTranslationExcellent(m) {
				excellent = append(excellent, m)
			} else {
				other = append(other, m)
			}
		}
		sort.Strings(excellent)
		sort.Strings(other)
		return append(excellent, other...)
	}
	var filtered []string
	for _, m := range all {
		if IsTranslationExcellent(m) {
			filtered = append(filtered, m)
		}
	}
	sort.Strings(filtered)
	return filtered
}

func (a *App) HasAPIKey(provider string) bool {
	if pc, ok := a.Config.Provider(provider); ok {
		return strings.TrimSpace(pc.APIKey) != ""
	}
	return false
}

func (a *App) SetProviderAPIKey(name, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("empty API key")
	}
	a.Config.SetAPIKey(name, key)
	if err := config.Save(a.Config); err != nil {
		return err
	}
	// rebuild provider instance
	pc, _ := a.Config.Provider(name)
	pcfg := provider.Config{
		Name:    pc.Name,
		BaseURL: pc.BaseURL,
		APIKey:  pc.APIKey,
		Model:   pc.Model,
		Org:     pc.Org,
		Project: pc.Project,
		Timeout: timeDuration(pc.Timeout),
	}
	p, err := provider.New(pcfg)
	if err != nil {
		return err
	}
	if a.Runtime.Providers == nil {
		a.Runtime.Providers = map[string]runtime.Provider{}
	}
	a.Runtime.Providers[name] = p
	return nil
}

func timeDuration(sec int) time.Duration {
	if sec <= 0 {
		sec = 60
	}
	return time.Duration(sec) * time.Second
}
