// Package language defines language identifiers and helpers.
package language

import "strings"

// Language is an ISO-ish language code used throughout tnow.
type Language string

const (
	Auto Language = "auto"
	EN   Language = "en"
	VI   Language = "vi"
	JA   Language = "ja"
	KO   Language = "ko"
	ZH   Language = "zh"
	FR   Language = "fr"
	ES   Language = "es"
	DE   Language = "de"
)

// Normalize lowercases and trims a language code.
func Normalize(s string) Language {
	return Language(strings.ToLower(strings.TrimSpace(s)))
}

// DisplayName returns a human-readable name.
func DisplayName(l Language) string {
	switch l {
	case Auto:
		return "Auto"
	case EN:
		return "English"
	case VI:
		return "Vietnamese"
	case JA:
		return "Japanese"
	case KO:
		return "Korean"
	case ZH:
		return "Chinese"
	case FR:
		return "French"
	case ES:
		return "Spanish"
	case DE:
		return "German"
	default:
		if s := string(l); s != "" {
			return s
		}
		return "Unknown"
	}
}

// IsKnown reports whether l is a recognized language.
func IsKnown(l Language) bool {
	switch l {
	case Auto, EN, VI, JA, KO, ZH, FR, ES, DE:
		return true
	}
	return false
}
