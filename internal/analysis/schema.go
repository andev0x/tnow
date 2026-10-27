package analysis

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/andev0x/tnow/internal/language"
)

// JSONSchema returns the strict instruction for the model.
func JSONSchema(target language.Language) string {
	return fmt.Sprintf(`Analyze the given word or short phrase and return JSON only.

Schema:
{
  "word": string,
  "language": string,
  "ipa": string,
  "part_of_speech": string,
  "definitions": [
    { "meaning": string, "translations": [string] }
  ],
  "examples": [
    { "source": string, "translation": string }
  ]
}

Rules:
- Return valid JSON only. No markdown fences.
- "translations" must be in %s.
- Provide 1-3 definitions and 1-3 examples.
- Keep meanings concise.`, language.DisplayName(target))
}

// BuildUserPrompt wraps the word.
func BuildUserPrompt(word string) string {
	return word
}

// Parse parses raw model output into Result, handling accidental fences.
func Parse(raw string) (Result, error) {
	s := strings.TrimSpace(raw)
	s = stripFences(s)
	var r Result
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return Result{}, err
	}
	return r, nil
}

func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// remove first fence line
		lines := strings.Split(s, "\n")
		if len(lines) > 0 {
			lines = lines[1:]
		}
		// remove trailing fence
		if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
			lines = lines[:len(lines)-1]
		}
		s = strings.Join(lines, "\n")
	}
	return strings.TrimSpace(s)
}
