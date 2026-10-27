package analysis

import "github.com/andev0x/tnow/internal/language"

type Request struct {
	Word       string
	SourceLang language.Language
	TargetLang language.Language
}

type Definition struct {
	Meaning      string   `json:"meaning"`
	Translations []string `json:"translations"`
}

type Example struct {
	Source      string `json:"source"`
	Translation string `json:"translation"`
}

type Result struct {
	Word         string       `json:"word"`
	Language     string       `json:"language"`
	IPA          string       `json:"ipa"`
	PartOfSpeech string       `json:"part_of_speech"`
	Definitions  []Definition `json:"definitions"`
	Examples     []Example    `json:"examples"`
}
