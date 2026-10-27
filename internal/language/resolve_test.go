package language

import "testing"

func TestResolveLanguageCountryAndCode(t *testing.T) {
	tests := []struct {
		input string
		want  Language
		ok    bool
	}{
		{"vi", VI, true},
		{"vie", VI, true},
		{"vietnamese", VI, true},
		{"vietnam", VI, true},
		{"Viet Nam", VI, true},
		{"tiếng việt", VI, true},
		{"japan", JA, true},
		{"japanese", JA, true},
		{"jp", JA, true},
		{"south korea", KO, true},
		{"korea", KO, true},
		{"kr", KO, true},
		{"united states", EN, true},
		{"USA", EN, true},
		{"uk", EN, true},
		{"england", EN, true},
		{"canada", EN, true},
		{"china", ZH, true},
		{"mandarin", ZH, true},
		{"taiwan", ZHTW, true},
		{"zh_tw", ZHTW, true},
		{"traditional chinese", ZHTW, true},
		{"hong kong", YUE, true},
		{"cantonese", YUE, true},
		{"brazil", PT, true},
		{"brasil", PT, true},
		{"portuguese", PT, true},
		{"mexico", ES, true},
		{"español", ES, true},
		{"france", FR, true},
		{"français", FR, true},
		{"germany", DE, true},
		{"deutschland", DE, true},
		{"russia", RU, true},
		{"india", HI, true},
		{"hindi", HI, true},
		{"saudi arabia", AR, true},
		{"arabic", AR, true},
		{"thailand", TH, true},
		{"indonesia", ID, true},
		{"malaysia", MS, true},
		{"myanmar", MY, true},
		{"cambodia", KM, true},
		{"laos", LO, true},
		{"italy", IT, true},
		{"netherlands", NL, true},
		{"czechia", CS, true},
		{"croatia", HR, true},
		{"hungary", HU, true},
		{"iceland", IS, true},
		{"ireland", GA, true},
		{"basque", EU, true},
		{"galicia", GL, true},
		{"tanzania", SW, true},
		{"haiti", HT, true},
		{"uzbekistan", UZ, true},
		{"kazakhstan", KK, true},
		{"kyrgyzstan", KY, true},
		{"việt nam", VI, true},
		{"nhật bản", JA, true},
		{"hàn quốc", KO, true},
		{"trung quốc", ZH, true},
		{"đài loan", ZHTW, true},
		{"pháp", FR, true},
		{"đức", DE, true},
		{"tây ban nha", ES, true},
		{"bồ đào nha", PT, true},
		{"thái lan", TH, true},
		{"ấn độ", HI, true},
		{"ả rập", AR, true},
		{"en-US", EN, true},
		{"en_GB", EN, true},
		{"pt-BR", PT, true},
		{"vi-VN", VI, true},
		{"zh-Hans", ZH, true},
		{"zh-Hant", ZHTW, true},
		{"auto", Auto, true},
		{"states", EN, true},
		{"united", Auto, false},
		{"e", Auto, false},
		{"nonsense", Auto, false},
		{"", Auto, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := Resolve(tt.input)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("Resolve(%q) = (%q, %v), want (%q, %v)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestSuggestRanksAndLabels(t *testing.T) {
	suggestions := Suggest("viet")
	if len(suggestions) == 0 || suggestions[0].Language != VI {
		t.Fatalf("Suggest(viet) = %+v, want Vietnamese first", suggestions)
	}
	if suggestions[0].Name != "Vietnamese" || suggestions[0].Region != "Vietnam" {
		t.Errorf("Vietnamese suggestion labels = %+v", suggestions[0])
	}

	all := Suggest("")
	if len(all) != len(All()) {
		t.Errorf("Suggest(\"\") returned %d languages, want %d", len(all), len(All()))
	}
	seen := make(map[Language]bool)
	for _, suggestion := range all {
		if seen[suggestion.Language] {
			t.Errorf("duplicate language in suggestions: %q", suggestion.Language)
		}
		seen[suggestion.Language] = true
	}

	korea := Suggest("korea")
	if len(korea) == 0 || korea[0].Language != KO {
		t.Fatalf("Suggest(korea) = %+v, want Korean first", korea)
	}

	// Country aliases should find the language even when the query is not a
	// prefix of the language code.
	mexico := Suggest("mexico")
	if len(mexico) == 0 || mexico[0].Language != ES {
		t.Fatalf("Suggest(mexico) = %+v, want Spanish first", mexico)
	}
}

func TestResolveIsCaseAndWhitespaceInsensitive(t *testing.T) {
	for _, input := range []string{"  SOUTH   KOREA ", "South Korea", "south-korea"} {
		got, ok := Resolve(input)
		if !ok || got != KO {
			t.Errorf("Resolve(%q) = (%q, %v), want Korean", input, got, ok)
		}
	}
}
