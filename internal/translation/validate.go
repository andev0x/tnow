package translation

import (
	"fmt"
	"strings"
)

// Normalize trims surrounding whitespace while preserving internal line breaks.
func Normalize(s string) string {
	// Remove a single trailing newline artifact; keep intentional blanks.
	s = strings.TrimRight(s, "\r\n")
	// Trim leading/trailing spaces on each? Preserve; but trim overall outer whitespace.
	s = strings.TrimSpace(s)
	return s
}

// Validate ensures the translation output is acceptable.
func Validate(raw string) (string, error) {
	out := Normalize(raw)
	if out == "" {
		return "", fmt.Errorf("empty translation output")
	}
	// Reject obvious conversational filler prefixes.
	lower := strings.ToLower(out)
	badPrefixes := []string{
		"here is the translation",
		"here's the translation",
		"sure!",
		"of course!",
		"certainly!",
	}
	for _, p := range badPrefixes {
		if strings.HasPrefix(lower, p) {
			return "", fmt.Errorf("invalid translation output: conversational filler")
		}
	}
	return out, nil
}
