package app

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// ModelInfo describes the useful, user-facing properties of a model. It is
// intentionally small: the model browser only needs the context window and the
// capabilities that affect translation and text processing.
type ModelInfo struct {
	ContextTokens int
	Vision        bool
	Tools         bool
	Known         bool
}

// explicitContext matches token windows written directly in a model name, such
// as "128k" or "1m". Version numbers like "gpt-4.1" and parameter counts like
// "7b" are deliberately not matched.
var explicitContext = regexp.MustCompile(`(?i)(^|[^a-z0-9])([0-9]+(?:\.[0-9]+)?)\s*([km])([^a-z0-9]|$)`)

// ModelInfoFor returns known context/capability information for a model. The
// lookup is heuristic: provider catalogs rarely expose a uniform metadata
// schema, so we combine explicit "128k"/"1m" markers in the model name with a
// small family catalog. Unknown models report Known=false instead of showing
// invented values.
func ModelInfoFor(model string) ModelInfo {
	name := strings.ToLower(strings.TrimSpace(model))
	if name == "" {
		return ModelInfo{}
	}

	info := ModelInfo{}
	if match := explicitContext.FindStringSubmatch(name); match != nil {
		if value, err := strconv.ParseFloat(match[2], 64); err == nil {
			switch strings.ToLower(match[3]) {
			case "k":
				info.ContextTokens = int(value * 1000)
			case "m":
				info.ContextTokens = int(value * 1_000_000)
			}
			info.Known = true
		}
	}

	has := func(parts ...string) bool {
		for _, part := range parts {
			if strings.Contains(name, part) {
				return true
			}
		}
		return false
	}
	visionHint := has("vision", "-vl", "vl-", "multimodal", "pixtral", "llava", "internvl", "minicpm-v")

	// Family rules, most specific first. A rule only fills context if the name
	// did not already contain an explicit size.
	setContext := func(tokens int) {
		if info.ContextTokens == 0 {
			info.ContextTokens = tokens
		}
	}
	switch {
	case has("gpt-5"):
		setContext(400_000)
		info.Vision, info.Tools, info.Known = true, true, true
	case has("gpt-4.1"):
		setContext(1_047_576)
		info.Vision, info.Tools, info.Known = true, true, true
	case has("gpt-4o", "gpt-4-turbo", "gpt-4-vision"):
		setContext(128_000)
		info.Vision, info.Tools, info.Known = true, true, true
	case has("gpt-4"):
		setContext(8_192)
		info.Vision, info.Tools, info.Known = has("vision"), true, true
	case has("gpt-3.5"):
		setContext(16_385)
		info.Tools, info.Known = true, true
	case has("o1", "o3", "o4"):
		setContext(200_000)
		info.Vision = has("o3", "o4")
		info.Tools, info.Known = true, true
	case has("claude"):
		setContext(200_000)
		info.Vision, info.Tools, info.Known = true, true, true
	case has("gemini"):
		setContext(1_048_576)
		info.Vision, info.Tools, info.Known = true, true, true
	case has("gemma-3", "gemma3"):
		setContext(131_072)
		info.Vision, info.Known = true, true
	case has("gemma"):
		setContext(8_192)
		info.Known = true
	case has("qwen"):
		setContext(32_768)
		info.Vision = visionHint
		info.Tools, info.Known = true, true
	case has("llama-4", "llama4"):
		setContext(1_048_576)
		info.Vision, info.Tools, info.Known = true, true, true
	case has("llama-3.2", "llama3.2"):
		setContext(131_072)
		info.Vision = has("vision")
		info.Tools, info.Known = true, true
	case has("llama-3", "llama3"):
		setContext(131_072)
		info.Tools, info.Known = true, true
	case has("llama"):
		setContext(8_192)
		info.Tools, info.Known = true, true
	case has("mistral", "mixtral", "magistral"):
		setContext(32_768)
		info.Vision = visionHint
		info.Tools, info.Known = true, true
	case has("deepseek"):
		setContext(65_536)
		info.Vision = visionHint
		info.Tools, info.Known = true, true
	case has("phi"):
		setContext(131_072)
		info.Tools, info.Known = true, true
	case has("command-r", "command-r+", "command-a"):
		setContext(131_072)
		info.Tools, info.Known = true, true
	case has("aya"):
		setContext(131_072)
		info.Known = true
	case has("allam"):
		setContext(131_072)
		info.Known = true
	case has("nova"):
		setContext(300_000)
		info.Vision, info.Tools, info.Known = true, true, true
	case has("grok"):
		setContext(131_072)
		info.Vision = visionHint || has("grok-4")
		info.Tools, info.Known = true, true
	case has("nllb"):
		setContext(512)
		info.Known = true
	case has("m2m"):
		setContext(1_024)
		info.Known = true
	case has("opus-mt", "seamless"):
		setContext(512)
		info.Known = true
	case has("whisper", "embed"):
		info.Known = true
	}

	if info.ContextTokens > 0 {
		info.Known = true
	}
	return info
}

// FormatContextTokens renders a token window in a compact, readable form.
func FormatContextTokens(tokens int) string {
	if tokens <= 0 {
		return "—"
	}
	if tokens >= 1_000_000 {
		value := float64(tokens) / 1_000_000
		if math.Abs(value-math.Round(value)) < 0.05 {
			return strconv.Itoa(int(math.Round(value))) + "M"
		}
		return strconv.FormatFloat(value, 'f', 1, 64) + "M"
	}
	if tokens%1000 == 0 {
		return strconv.Itoa(tokens/1000) + "k"
	}
	if tokens%1024 == 0 {
		return strconv.Itoa(tokens/1024) + "k"
	}
	if tokens >= 1_000 {
		return strconv.Itoa(int(math.Round(float64(tokens)/1_000))) + "k"
	}
	return strconv.Itoa(tokens)
}
