package analysis

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/andev0x/tnow/internal/language"
	"github.com/andev0x/tnow/internal/runtime"
)

type mockProvider struct {
	res     runtime.GenerationResult
	err     error
	lastReq runtime.GenerationRequest
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Generate(ctx context.Context, req runtime.GenerationRequest) (runtime.GenerationResult, error) {
	m.lastReq = req
	return m.res, m.err
}

func TestParseValid(t *testing.T) {
	raw := `{"word":"awkward","language":"en","ipa":"/x/","part_of_speech":"adjective","definitions":[{"meaning":"m","translations":["v"]}], "examples":[{"source":"s","translation":"t"}]}`
	r, err := Parse(raw)
	if err != nil || r.Word != "awkward" {
		t.Fatalf("got %v %v", r, err)
	}
}

func TestParseFences(t *testing.T) {
	raw := "```json\n{\"word\":\"w\",\"language\":\"en\",\"ipa\":\"\",\"part_of_speech\":\"noun\",\"definitions\":[{\"meaning\":\"m\",\"translations\":[]}],\"examples\":[]}\n```"
	r, err := Parse(raw)
	if err != nil || r.Word != "w" {
		t.Fatalf("got %v %v", r, err)
	}
}

func TestParseMalformed(t *testing.T) {
	if _, err := Parse("{bad"); err == nil {
		t.Fatal("expected err")
	}
}

func TestValidateMissingDefs(t *testing.T) {
	r := Result{Word: "w"}
	if err := Validate(r); err == nil {
		t.Fatal("expected err")
	}
}

func TestValidateOK(t *testing.T) {
	r := Result{Word: "w", Definitions: []Definition{{Meaning: "m"}}}
	if err := Validate(r); err != nil {
		t.Fatal(err)
	}
}

func TestEngineAnalyze(t *testing.T) {
	resObj := Result{Word: "awkward", Language: "en", IPA: "/x/", PartOfSpeech: "adjective", Definitions: []Definition{{Meaning: "m", Translations: []string{"v"}}}, Examples: []Example{{Source: "s", Translation: "t"}}}
	b, _ := json.Marshal(resObj)
	mp := &mockProvider{res: runtime.GenerationResult{Text: string(b)}}
	rt := runtime.New(map[string]runtime.Provider{"mock": mp}, "mock")
	eng := NewEngine(rt, "mock", "m")
	r, err := eng.Analyze(context.Background(), Request{Word: "awkward", TargetLang: language.VI})
	if err != nil || r.Word != "awkward" {
		t.Fatalf("got %v %v", r, err)
	}
}

func TestEngineAnalyzeDefaultsToEnglish(t *testing.T) {
	resObj := Result{Word: "awkward", Language: "en", IPA: "/x/", PartOfSpeech: "adjective", Definitions: []Definition{{Meaning: "m", Translations: []string{"v"}}}, Examples: []Example{{Source: "s", Translation: "t"}}}
	b, _ := json.Marshal(resObj)
	mp := &mockProvider{res: runtime.GenerationResult{Text: string(b)}}
	rt := runtime.New(map[string]runtime.Provider{"mock": mp}, "mock")
	eng := NewEngine(rt, "mock", "m")
	if _, err := eng.Analyze(context.Background(), Request{Word: "awkward"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mp.lastReq.SystemPrompt, "English") {
		t.Fatalf("default target prompt = %q, want English", mp.lastReq.SystemPrompt)
	}
}

func TestJSONSchemaNamesBothLanguages(t *testing.T) {
	prompt := JSONSchema(language.VI, language.EN)
	for _, want := range []string{
		"Input language: Vietnamese",
		"Output language: English",
		`Set "language" to "Vietnamese"`,
		`"examples[].source" as a natural Vietnamese sentence`,
		`"examples[].translation" in English`,
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("schema prompt missing %q:\n%s", want, prompt)
		}
	}

	auto := JSONSchema(language.Auto, language.EN)
	if !strings.Contains(auto, "Detect the input language") {
		t.Errorf("auto schema prompt should ask the model to detect the input language:\n%s", auto)
	}
}

func TestEngineAnalyzeUsesDetectedSourceLanguage(t *testing.T) {
	// Simulate a model that mislabels Vietnamese as Chinese and answers in
	// Chinese even though the requested target is English.
	resObj := Result{
		Word:         "xin chào",
		Language:     "Chinese",
		IPA:          "/ɕin tɕaʊ/",
		PartOfSpeech: "interjection",
		Definitions:  []Definition{{Meaning: "Hello", Translations: []string{"Hello"}}},
		Examples:     []Example{{Source: "When you meet someone", Translation: "你好 (nǐ hǎo)"}},
	}
	b, _ := json.Marshal(resObj)
	mp := &mockProvider{res: runtime.GenerationResult{Text: string(b)}}
	rt := runtime.New(map[string]runtime.Provider{"mock": mp}, "mock")
	eng := NewEngine(rt, "mock", "m")

	r, err := eng.Analyze(context.Background(), Request{
		Word:       "xin chào",
		SourceLang: language.Auto,
		TargetLang: language.EN,
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Language != "Vietnamese" {
		t.Errorf("language = %q, want Vietnamese", r.Language)
	}
	if !strings.Contains(mp.lastReq.SystemPrompt, "Input language: Vietnamese") {
		t.Errorf("prompt did not name the detected input language:\n%s", mp.lastReq.SystemPrompt)
	}
	if !strings.Contains(mp.lastReq.SystemPrompt, "Output language: English") {
		t.Errorf("prompt did not name the output language:\n%s", mp.lastReq.SystemPrompt)
	}
}

func TestEngineAnalyzeHonorsExplicitSourceLanguage(t *testing.T) {
	resObj := Result{Word: "hola", Language: "Chinese", Definitions: []Definition{{Meaning: "hello", Translations: []string{"hello"}}}}
	b, _ := json.Marshal(resObj)
	mp := &mockProvider{res: runtime.GenerationResult{Text: string(b)}}
	rt := runtime.New(map[string]runtime.Provider{"mock": mp}, "mock")
	eng := NewEngine(rt, "mock", "m")

	r, err := eng.Analyze(context.Background(), Request{
		Word:       "hola",
		SourceLang: language.ES,
		TargetLang: language.EN,
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Language != "Spanish" {
		t.Errorf("language = %q, want Spanish", r.Language)
	}
	if !strings.Contains(mp.lastReq.SystemPrompt, "Input language: Spanish") {
		t.Errorf("prompt did not name the explicit input language:\n%s", mp.lastReq.SystemPrompt)
	}
}

func TestEngineAnalyzeMalformed(t *testing.T) {
	mp := &mockProvider{res: runtime.GenerationResult{Text: "not json"}}
	rt := runtime.New(map[string]runtime.Provider{"mock": mp}, "mock")
	eng := NewEngine(rt, "mock", "m")
	_, err := eng.Analyze(context.Background(), Request{Word: "awkward", TargetLang: language.VI})
	if err == nil {
		t.Fatal("expected err")
	}
}
