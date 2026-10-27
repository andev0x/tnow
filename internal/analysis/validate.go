package analysis

import (
	"fmt"
	"strings"
)

// Validate checks a parsed analysis result.
func Validate(r Result) error {
	if strings.TrimSpace(r.Word) == "" {
		return fmt.Errorf("missing word")
	}
	_ = strings.TrimSpace(r.Language) // optional, filled later
	if len(r.Definitions) == 0 {
		return fmt.Errorf("missing definitions")
	}
	for i, d := range r.Definitions {
		if strings.TrimSpace(d.Meaning) == "" {
			return fmt.Errorf("definition %d missing meaning", i)
		}
	}
	// examples optional but validate if present
	for i, e := range r.Examples {
		if strings.TrimSpace(e.Source) == "" || strings.TrimSpace(e.Translation) == "" {
			return fmt.Errorf("example %d invalid", i)
		}
	}
	return nil
}
