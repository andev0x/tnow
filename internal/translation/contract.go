package translation

import (
	"fmt"

	"github.com/andev0x/tnow/internal/language"
	"github.com/andev0x/tnow/internal/runtime"
)

// SystemPrompt builds the minimal translation instruction.
func SystemPrompt(target language.Language) string {
	return fmt.Sprintf(`Translate the input into %s.

Return only the translation.

Preserve:
- meaning
- tone
- formatting
- line breaks

Do not explain.
Do not add commentary.
Do not prefix the translation.`, language.DisplayName(target))
}

// BuildRequest converts a domain request into a bounded generation request.
func BuildRequest(req Request, provider, model string, maxTokens int, temperature float64, stream bool) runtime.GenerationRequest {
	src := req.SourceLang
	_ = src // auto-detection optional, leave as-is when empty or Auto
	user := req.Text
	if req.Context != "" {
		user = req.Context + "\n\n" + req.Text
	}
	if maxTokens <= 0 {
		maxTokens = 2048
	}
	if temperature < 0 {
		temperature = 0.2
	}
	return runtime.GenerationRequest{
		Provider:     provider,
		Model:        model,
		SystemPrompt: SystemPrompt(req.TargetLang),
		UserPrompt:   user,
		MaxTokens:    maxTokens,
		Temperature:  temperature,
		Stream:       stream,
		Operation:    "translation",
	}
}
