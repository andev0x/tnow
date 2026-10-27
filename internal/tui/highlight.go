package tui

import (
	"regexp"
	"strings"

	"github.com/andev0x/tnow/internal/analysis"
)

var (
	reHighlight = regexp.MustCompile(`==([^=]+)==`)
	reBoldStar  = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reCodeTick  = regexp.MustCompile("`([^`]+)`")
)

// HighlightMatches highlights all case-insensitive occurrences of query in text.
func HighlightMatches(text, query string, s Styles) string {
	if query == "" || text == "" {
		return text
	}

	qLower := strings.ToLower(query)
	var b strings.Builder
	remaining := text

	for {
		idx := strings.Index(strings.ToLower(remaining), qLower)
		if idx == -1 {
			b.WriteString(remaining)
			break
		}
		// Write prefix
		b.WriteString(remaining[:idx])
		// Write highlighted match
		match := remaining[idx : idx+len(query)]
		b.WriteString(s.HighlightMatch.Render(match))
		remaining = remaining[idx+len(query):]
	}

	return b.String()
}

// RenderHighlightedText parses markdown/highlight markers and query matches.
func RenderHighlightedText(text, query string, s Styles) string {
	if text == "" {
		return ""
	}

	// 1. Process line by line for block/section elements
	lines := strings.Split(text, "\n")
	var processedLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Heading or section title
		if strings.HasPrefix(trimmed, "### ") {
			title := strings.TrimPrefix(trimmed, "### ")
			line = s.SectionTitle.Render("◆ " + title)
		} else if strings.HasPrefix(trimmed, "## ") {
			title := strings.TrimPrefix(trimmed, "## ")
			line = s.SectionTitle.Render("◆ " + title)
		} else if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimPrefix(trimmed, "# ")
			line = s.SectionTitle.Render("■ " + title)
		} else if strings.HasPrefix(trimmed, "• ") || strings.HasPrefix(trimmed, "- ") {
			bullet := s.BulletPoint.Render("•")
			rest := trimmed[2:]
			line = "  " + bullet + " " + rest
		}

		// Replace inline markdown highlights: ==text==
		line = reHighlight.ReplaceAllStringFunc(line, func(match string) string {
			content := match[2 : len(match)-2]
			return s.Highlight.Render(content)
		})

		// Replace inline bold: **text**
		line = reBoldStar.ReplaceAllStringFunc(line, func(match string) string {
			content := match[2 : len(match)-2]
			return s.HighlightBold.Render(content)
		})

		// Replace inline code/tags: `code`
		line = reCodeTick.ReplaceAllStringFunc(line, func(match string) string {
			content := match[1 : len(match)-1]
			return s.HighlightCode.Render(content)
		})

		// Arrow highlights
		line = strings.ReplaceAll(line, "→", s.ArrowSymbol.Render("➜"))
		line = strings.ReplaceAll(line, "->", s.ArrowSymbol.Render("➜"))

		// If there is an active search query, highlight matching text
		if query != "" {
			line = HighlightMatches(line, query, s)
		}

		processedLines = append(processedLines, line)
	}

	return strings.Join(processedLines, "\n")
}

// RenderAnalysisStructured renders analysis result with structured section highlights.
func RenderAnalysisStructured(r analysis.Result, query string, s Styles, width int) string {
	var b strings.Builder

	// Top Section: Word, Phonetics, and Part of Speech
	wordStyled := s.HighlightBold.Foreground(s.Logo.GetForeground()).Render(r.Word)
	b.WriteString(wordStyled)

	var badges []string
	if r.IPA != "" {
		ipaText := r.IPA
		if !strings.HasPrefix(ipaText, "/") && !strings.HasPrefix(ipaText, "[") {
			ipaText = "/" + ipaText + "/"
		}
		badges = append(badges, s.PhoneticTag.Render(ipaText))
	}
	if r.PartOfSpeech != "" {
		badges = append(badges, s.PosBadge.Render(strings.ToUpper(r.PartOfSpeech)))
	}
	if r.Language != "" {
		badges = append(badges, s.BadgePill.Render(strings.ToUpper(r.Language)))
	}

	if len(badges) > 0 {
		b.WriteString("  ")
		b.WriteString(strings.Join(badges, " "))
	}
	b.WriteString("\n")

	// Subtle divider line
	dividerWidth := width - 6
	if dividerWidth <= 0 {
		dividerWidth = 40
	}
	if dividerWidth > 70 {
		dividerWidth = 70
	}
	b.WriteString(s.SubtleLine.Render(strings.Repeat("─", dividerWidth)))
	b.WriteString("\n\n")

	// Section 1: Meanings / Definitions
	if len(r.Definitions) > 0 {
		b.WriteString(s.SectionTitle.Render("◆ MEANINGS"))
		b.WriteString("\n\n")

		for _, d := range r.Definitions {
			b.WriteString("  " + s.BulletPoint.Render("•") + " ")
			b.WriteString(d.Meaning)
			b.WriteString("\n")

			if len(d.Translations) > 0 {
				b.WriteString("    " + s.ArrowSymbol.Render("➜") + " ")
				b.WriteString(s.TranslationWord.Render(strings.Join(d.Translations, ", ")))
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	// Section 2: Examples
	if len(r.Examples) > 0 {
		b.WriteString(s.SectionTitle.Render("◆ EXAMPLES"))
		b.WriteString("\n\n")

		for _, e := range r.Examples {
			b.WriteString("  " + s.ExampleSource.Render("“"+e.Source+"”"))
			b.WriteString("\n")
			b.WriteString("    " + s.ArrowSymbol.Render("➜") + " ")
			b.WriteString(s.ExampleTarget.Render(e.Translation))
			b.WriteString("\n\n")
		}
	}

	raw := strings.TrimRight(b.String(), "\n")

	// Apply query highlighting if search/highlight query is present
	if query != "" {
		raw = HighlightMatches(raw, query, s)
	}

	return raw
}
