package analysis

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/andev0x/tnow/internal/language"
)

// JSONSchema returns the strict instruction for the model. Both languages are
// named explicitly: without a source language the model can mistake a
// Vietnamese word for Chinese and answer in the wrong language.
func JSONSchema(source, target language.Language) string {
	output := language.DisplayName(target)

	input := "auto-detect from the input"
	inputRules := `- Detect the input language and set "language" to its English name.
- "ipa" must describe the input word in that language.
- Write each "examples[].source" in that language.`
	if source != "" && source != language.Auto {
		input = language.DisplayName(source)
		inputRules = fmt.Sprintf(`- The input is %s. Set "language" to %q.
- "ipa" must describe the input word in %s.
- Write each "examples[].source" as a natural %s sentence that uses the input.`,
			input, input, input, input)
	}

	return fmt.Sprintf(`Analyze the given word or short phrase and return JSON only.

Input language: %s
Output language: %s

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
- "language" names the input language, never the output language.
%s
- Write every definition "meaning", every "translations" value, and every
  "examples[].translation" in %s.
- Keep the input and output languages distinct. Never substitute an unrelated
  language for the input or the output.
- Provide 1-3 definitions and 1-3 examples.
- Keep meanings concise.`, input, output, inputRules, output)
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
