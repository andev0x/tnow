package analysis

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/andev0x/tnow/internal/language"
	"github.com/andev0x/tnow/internal/runtime"
)

type mockProvider struct {
	res runtime.GenerationResult
	err error
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Generate(ctx context.Context, req runtime.GenerationRequest) (runtime.GenerationResult, error) {
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

func TestEngineAnalyzeMalformed(t *testing.T) {
	mp := &mockProvider{res: runtime.GenerationResult{Text: "not json"}}
	rt := runtime.New(map[string]runtime.Provider{"mock": mp}, "mock")
	eng := NewEngine(rt, "mock", "m")
	_, err := eng.Analyze(context.Background(), Request{Word: "awkward", TargetLang: language.VI})
	if err == nil {
		t.Fatal("expected err")
	}
}
