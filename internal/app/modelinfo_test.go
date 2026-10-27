package app

import "testing"

func TestModelInfoFor(t *testing.T) {
	tests := []struct {
		model         string
		wantContext   int
		wantVision    bool
		wantTools     bool
		wantKnown     bool
		wantFormatted string
	}{
		{"openai/gpt-4o", 128_000, true, true, true, "128k"},
		{"gpt-4.1-mini", 1_047_576, true, true, true, "1M"},
		{"claude-sonnet-4", 200_000, true, true, true, "200k"},
		{"gemini-2.5-pro", 1_048_576, true, true, true, "1M"},
		{"llama-3.3-70b-versatile", 131_072, false, true, true, "128k"},
		{"qwen2.5-72b-instruct", 32_768, false, true, true, "32k"},
		{"allam-2-7b", 131_072, false, false, true, "128k"},
		{"some-unknown-model", 0, false, false, false, "—"},
	}
	for _, tt := range tests {
		info := ModelInfoFor(tt.model)
		if info.ContextTokens != tt.wantContext || info.Vision != tt.wantVision || info.Tools != tt.wantTools || info.Known != tt.wantKnown {
			t.Errorf("ModelInfoFor(%q) = %+v, want context=%d vision=%v tools=%v known=%v",
				tt.model, info, tt.wantContext, tt.wantVision, tt.wantTools, tt.wantKnown)
		}
		if got := FormatContextTokens(info.ContextTokens); got != tt.wantFormatted {
			t.Errorf("FormatContextTokens(%d) = %q, want %q", info.ContextTokens, got, tt.wantFormatted)
		}
	}
}

func TestFormatContextTokens(t *testing.T) {
	cases := map[int]string{
		0:         "—",
		512:       "512",
		8_192:     "8k",
		32_768:    "32k",
		128_000:   "128k",
		131_072:   "128k",
		1_048_576: "1M",
	}
	for tokens, want := range cases {
		if got := FormatContextTokens(tokens); got != want {
			t.Errorf("FormatContextTokens(%d) = %q, want %q", tokens, got, want)
		}
	}
}
