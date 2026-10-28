package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andev0x/tnow/internal/analysis"
	"github.com/andev0x/tnow/internal/app"
	"github.com/andev0x/tnow/internal/language"
	"github.com/andev0x/tnow/internal/translation"
)

// Run executes CLI mode. Returns exit code. -1 means launch TUI.
func Run(a *app.App, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tnow", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var from, to, model, provider string
	var word, verbose, debug bool
	fs.StringVar(&from, "from", "", "source language")
	fs.StringVar(&from, "f", "", "source language")
	fs.StringVar(&to, "to", "", "target language")
	fs.StringVar(&to, "t", "", "target language")
	fs.BoolVar(&word, "word", false, "word analysis mode")
	fs.BoolVar(&word, "w", false, "word analysis mode")
	fs.StringVar(&model, "model", "", "model")
	fs.StringVar(&model, "m", "", "model")
	fs.StringVar(&provider, "provider", "", "provider")
	fs.StringVar(&provider, "p", "", "provider")
	fs.BoolVar(&verbose, "verbose", false, "verbose")
	fs.BoolVar(&debug, "debug", false, "debug")
	_ = verbose
	_ = debug
	_ = os.Stderr

	// Reorder so flags can appear after positional text.
	args = reorderArgs(args)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if provider != "" {
		a.SetProvider(provider)
	}
	if model != "" {
		a.SetModel(model)
	}
	if from != "" {
		a.SetSource(resolveLanguage(from))
	}
	if to != "" {
		a.SetTarget(resolveLanguage(to))
	}

	text := strings.Join(fs.Args(), " ")
	if strings.TrimSpace(text) == "" && stdin != nil {
		sc := bufio.NewScanner(stdin)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024)
		var lines []string
		for sc.Scan() {
			lines = append(lines, sc.Text())
		}
		text = strings.Join(lines, "\n")
	}
	text = strings.TrimSpace(text)

	if text == "" {
		return -1
	}

	ctx := context.Background()

	if word {
		res, err := a.Analysis.Analyze(ctx, analysis.Request{
			Word:       text,
			SourceLang: a.SourceLang,
			TargetLang: a.TargetLang,
		})
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return 1
		}
		renderAnalysis(stdout, res)
		return 0
	}

	req := translation.Request{
		Text:       text,
		SourceLang: a.SourceLang,
		TargetLang: a.TargetLang,
	}
	res, err := a.Translation.Translate(ctx, req)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 1
	}
	_, _ = fmt.Fprintln(stdout, res.Text)
	return 0
}

func resolveLanguage(value string) language.Language {
	if lang, ok := language.Resolve(value); ok {
		return lang
	}
	return language.Normalize(value)
}

func reorderArgs(args []string) []string {
	valueFlags := map[string]bool{
		"from": true, "f": true, "to": true, "t": true,
		"model": true, "m": true, "provider": true, "p": true,
	}
	var flags, pos []string
	i := 0
	for i < len(args) {
		a := args[i]
		name := strings.TrimLeft(a, "-")
		if idx := strings.Index(name, "="); idx >= 0 {
			name = name[:idx]
		}
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && valueFlags[name] && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i += 2
				continue
			}
			i++
			continue
		}
		pos = append(pos, a)
		i++
	}
	return append(flags, pos...)
}

func renderAnalysis(w io.Writer, r analysis.Result) {
	_, _ = fmt.Fprintln(w, r.Word)
	_, _ = fmt.Fprintln(w, strings.Repeat("─", 28))
	_, _ = fmt.Fprintln(w)
	if r.IPA != "" {
		_, _ = fmt.Fprintln(w, r.IPA)
		_, _ = fmt.Fprintln(w)
	}
	if r.PartOfSpeech != "" {
		_, _ = fmt.Fprintln(w, strings.ToUpper(r.PartOfSpeech))
		_, _ = fmt.Fprintln(w)
	}
	_, _ = fmt.Fprintln(w, "Meanings")
	_, _ = fmt.Fprintln(w)
	for _, d := range r.Definitions {
		_, _ = fmt.Fprintf(w, "• %s\n", d.Meaning)
		if len(d.Translations) > 0 {
			_, _ = fmt.Fprintf(w, "  %s\n", strings.Join(d.Translations, ", "))
		}
	}
	if len(r.Examples) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Examples")
		_, _ = fmt.Fprintln(w)
		for _, e := range r.Examples {
			_, _ = fmt.Fprintf(w, "%s\n→ %s\n\n", e.Source, e.Translation)
		}
	}
}
