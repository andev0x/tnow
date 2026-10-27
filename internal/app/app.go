package app

import (
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
	if pc, ok := a.Config.Provider(name); ok {
		if pc.Model != "" && a.Model == "" {
			a.Model = pc.Model
		}
	}
	a.Translation.Provider = a.Provider
	a.Translation.Model = a.Model
	a.Analysis.Provider = a.Provider
	a.Analysis.Model = a.Model
}

func (a *App) SetModel(model string) {
	a.Model = model
	a.Translation.Model = model
	a.Analysis.Model = model
}

func (a *App) SetTarget(lang language.Language) {
	a.TargetLang = lang
}

func (a *App) SetSource(lang language.Language) {
	a.SourceLang = lang
}

func timeDuration(sec int) time.Duration {
	if sec <= 0 {
		sec = 60
	}
	return time.Duration(sec) * time.Second
}
