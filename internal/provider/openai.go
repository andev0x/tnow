package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/andev0x/tnow/internal/runtime"
)

type OpenAICompatible struct {
	cfg    Config
	client *http.Client
}

func NewOpenAICompatible(cfg Config) *OpenAICompatible {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &OpenAICompatible{cfg: cfg, client: &http.Client{Timeout: timeout}}
}

func (p *OpenAICompatible) Name() string { return p.cfg.Name }

// ListModels returns the models exposed by an OpenAI-compatible endpoint.
func (p *OpenAICompatible) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.BaseURL+"/models", nil)
	if err != nil {
		return nil, &runtime.RuntimeError{Kind: runtime.ErrNetwork, Err: err}
	}
	if p.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	}
	if p.cfg.Org != "" {
		req.Header.Set("OpenAI-Organization", p.cfg.Org)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		kind := runtime.ErrNetwork
		if ctx.Err() != nil {
			if ctx.Err() == context.Canceled {
				kind = runtime.ErrCancellation
			} else {
				kind = runtime.ErrTimeout
			}
		}
		return nil, &runtime.RuntimeError{Kind: kind, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		kind := runtime.ErrModel
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			kind = runtime.ErrAuth
		}
		return nil, &runtime.RuntimeError{Kind: kind, Err: fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))}
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, &runtime.RuntimeError{Kind: runtime.ErrModel, Err: err}
	}

	seen := make(map[string]bool)
	var models []string
	for _, m := range payload.Data {
		if m.ID == "" || seen[m.ID] {
			continue
		}
		seen[m.ID] = true
		models = append(models, m.ID)
	}
	return models, nil
}

type oaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaiRequest struct {
	Model       string       `json:"model"`
	Messages    []oaiMessage `json:"messages"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Temperature float64      `json:"temperature"`
	Stream      bool         `json:"stream,omitempty"`
}

type oaiChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type oaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type oaiResponse struct {
	Choices []oaiChoice `json:"choices"`
	Usage   oaiUsage    `json:"usage"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (p *OpenAICompatible) Generate(ctx context.Context, req runtime.GenerationRequest) (runtime.GenerationResult, error) {
	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}
	body := oaiRequest{
		Model: model,
		Messages: []oaiMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserPrompt},
		},
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      req.Stream,
	}
	b, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.cfg.BaseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return runtime.GenerationResult{}, &runtime.RuntimeError{Kind: runtime.ErrNetwork, Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	}
	if p.cfg.Org != "" {
		httpReq.Header.Set("OpenAI-Organization", p.cfg.Org)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		kind := runtime.ErrNetwork
		if ctx.Err() != nil {
			if ctx.Err() == context.Canceled {
				kind = runtime.ErrCancellation
			} else {
				kind = runtime.ErrTimeout
			}
		}
		return runtime.GenerationResult{}, &runtime.RuntimeError{Kind: kind, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		kind := runtime.ErrModel
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			kind = runtime.ErrAuth
		}
		if resp.StatusCode == 429 {
			kind = runtime.ErrRateLimit
		}
		if resp.StatusCode >= 500 {
			kind = runtime.ErrProviderUnavail
		}
		return runtime.GenerationResult{}, &runtime.RuntimeError{Kind: kind, Err: fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))}
	}

	if req.Stream {
		ch := make(chan runtime.StreamEvent, 32)
		go func() {
			defer close(ch)
			scanner := bufio.NewScanner(resp.Body)
			scanner.Buffer(make([]byte, 64*1024), 1024*1024)
			var usage runtime.Usage
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if data == "[DONE]" {
					ch <- runtime.StreamEvent{Done: true, Usage: usage}
					return
				}
				var r oaiResponse
				if err := json.Unmarshal([]byte(data), &r); err != nil {
					continue
				}
				if r.Error != nil {
					ch <- runtime.StreamEvent{Err: &runtime.RuntimeError{Kind: runtime.ErrModel, Err: fmt.Errorf("%s", r.Error.Message)}}
					return
				}
				if len(r.Choices) > 0 {
					if c := r.Choices[0].Delta.Content; c != "" {
						ch <- runtime.StreamEvent{Chunk: c}
					}
				}
				if r.Usage.TotalTokens > 0 {
					usage = runtime.Usage{PromptTokens: r.Usage.PromptTokens, CompletionTokens: r.Usage.CompletionTokens, TotalTokens: r.Usage.TotalTokens}
				}
			}
			if err := scanner.Err(); err != nil {
				ch <- runtime.StreamEvent{Err: &runtime.RuntimeError{Kind: runtime.ErrNetwork, Err: err}}
				return
			}
			ch <- runtime.StreamEvent{Done: true, Usage: usage}
		}()
		return runtime.GenerationResult{Stream: ch}, nil
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return runtime.GenerationResult{}, &runtime.RuntimeError{Kind: runtime.ErrNetwork, Err: err}
	}
	var r oaiResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return runtime.GenerationResult{}, &runtime.RuntimeError{Kind: runtime.ErrModel, Err: err}
	}
	if r.Error != nil {
		return runtime.GenerationResult{}, &runtime.RuntimeError{Kind: runtime.ErrModel, Err: fmt.Errorf("%s", r.Error.Message)}
	}
	if len(r.Choices) == 0 {
		return runtime.GenerationResult{}, &runtime.RuntimeError{Kind: runtime.ErrModel, Err: fmt.Errorf("no choices returned")}
	}
	text := r.Choices[0].Message.Content
	usage := runtime.Usage{PromptTokens: r.Usage.PromptTokens, CompletionTokens: r.Usage.CompletionTokens, TotalTokens: r.Usage.TotalTokens}
	return runtime.GenerationResult{Text: text, Usage: usage}, nil
}
