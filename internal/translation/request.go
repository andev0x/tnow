package translation

import "github.com/andev0x/tnow/internal/language"

type Request struct {
	Text       string
	SourceLang language.Language
	TargetLang language.Language
	Context    string
}

type Result struct {
	Text       string
	SourceLang language.Language
	TargetLang language.Language
}
