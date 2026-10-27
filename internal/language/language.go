// Package language defines language identifiers and helpers.
package language

import "strings"

// Language is an ISO-ish language code used throughout tnow.
type Language string

const (
	Auto Language = "auto"

	// Major global & high-resource languages.
	EN   Language = "en"
	ES   Language = "es"
	FR   Language = "fr"
	DE   Language = "de"
	ZH   Language = "zh"    // Chinese (Simplified)
	ZHTW Language = "zh-tw" // Chinese (Traditional)
	JA   Language = "ja"
	KO   Language = "ko"
	PT   Language = "pt"
	RU   Language = "ru"

	// Asian & Middle Eastern languages.
	VI  Language = "vi"
	HI  Language = "hi"
	AR  Language = "ar"
	TH  Language = "th"
	YUE Language = "yue" // Cantonese
	ID  Language = "id"  // Indonesian
	MS  Language = "ms"  // Malay
	MY  Language = "my"  // Burmese
	KM  Language = "km"  // Khmer
	LO  Language = "lo"  // Lao

	// European & regional languages.
	IT Language = "it"
	NL Language = "nl"
	CS Language = "cs"
	HR Language = "hr"
	HU Language = "hu"
	IS Language = "is"
	GA Language = "ga"
	EU Language = "eu" // Basque
	GL Language = "gl" // Galician

	// African & low-resource languages.
	SW Language = "sw"
	HT Language = "ht" // Haitian Creole
	UZ Language = "uz"
	KK Language = "kk"
	KY Language = "ky"
)

// Common aliases for multi-part / alternative spellings.
const (
	ZH_TW              = ZHTW
	ZH_HANS            = ZH
	ZH_HANT            = ZHTW
	ZH_TRADITIONAL     = ZHTW
	ZH_SIMPLIFIED      = ZH
	Cantonese          = YUE
	Indonesian         = ID
	Malay              = MS
	Burmese            = MY
	Khmer              = KM
	Lao                = LO
	TraditionalChinese = ZHTW
	SimplifiedChinese  = ZH
)

// displayNames maps a normalized (lower-case) language code to a human-readable name.
var displayNames = map[string]string{
	"auto":  "Auto",
	"en":    "English",
	"es":    "Spanish",
	"fr":    "French",
	"de":    "German",
	"zh":    "Chinese",
	"zh-tw": "Chinese (Traditional)",
	"ja":    "Japanese",
	"ko":    "Korean",
	"pt":    "Portuguese",
	"ru":    "Russian",
	"vi":    "Vietnamese",
	"hi":    "Hindi",
	"ar":    "Arabic",
	"th":    "Thai",
	"yue":   "Cantonese",
	"id":    "Indonesian",
	"ms":    "Malay",
	"my":    "Burmese",
	"km":    "Khmer",
	"lo":    "Lao",
	"it":    "Italian",
	"nl":    "Dutch",
	"cs":    "Czech",
	"hr":    "Croatian",
	"hu":    "Hungarian",
	"is":    "Icelandic",
	"ga":    "Irish",
	"eu":    "Basque",
	"gl":    "Galician",
	"sw":    "Swahili",
	"ht":    "Haitian Creole",
	"uz":    "Uzbek",
	"kk":    "Kazakh",
	"ky":    "Kyrgyz",
}

// Normalize lowercases and trims a language code.
func Normalize(s string) Language {
	return Language(strings.ToLower(strings.TrimSpace(s)))
}

// DisplayName returns a human-readable name.
func DisplayName(l Language) string {
	key := strings.ToLower(strings.TrimSpace(string(l)))
	if name, ok := displayNames[key]; ok {
		return name
	}
	if s := strings.TrimSpace(string(l)); s != "" {
		return s
	}
	return "Unknown"
}

// IsKnown reports whether l is a recognized language.
func IsKnown(l Language) bool {
	key := strings.ToLower(strings.TrimSpace(string(l)))
	_, ok := displayNames[key]
	return ok
}

// All returns all supported languages (excluding Auto) in a stable order.
func All() []Language {
	return []Language{
		EN, ES, FR, DE, ZH, ZHTW, JA, KO, PT, RU,
		VI, HI, AR, TH, YUE, ID, MS, MY, KM, LO,
		IT, NL, CS, HR, HU, IS, GA, EU, GL,
		SW, HT, UZ, KK, KY,
	}
}

// AllLanguages is an alias for All.
func AllLanguages() []Language { return All() }

// Languages is an alias for All.
func Languages() []Language { return All() }

// Supported is an alias for All.
func Supported() []Language { return All() }

// SupportedLanguages is an alias for All.
func SupportedLanguages() []Language { return All() }
