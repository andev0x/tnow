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

type Ollama struct {
	cfg    Config
	client *http.Client
}

func NewOllama(cfg Config) *Ollama {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &Ollama{cfg: cfg, client: &http.Client{Timeout: timeout}}
}

func (p *Ollama) Name() string { return "ollama" }

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
	// usage
	PromptEvalCount int    `json:"prompt_eval_count"`
	EvalCount       int    `json:"eval_count"`
	Error           string `json:"error"`
}

func (p *Ollama) Generate(ctx context.Context, req runtime.GenerationRequest) (runtime.GenerationResult, error) {
	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}
	opts := map[string]any{}
	if req.MaxTokens > 0 {
		opts["num_predict"] = req.MaxTokens
	}
	opts["temperature"] = req.Temperature
	body := ollamaRequest{
		Model: model,
		Messages: []ollamaMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserPrompt},
		},
		Stream:  req.Stream,
		Options: opts,
	}
	b, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.cfg.BaseURL+"/api/chat", bytes.NewReader(b))
	if err != nil {
		return runtime.GenerationResult{}, &runtime.RuntimeError{Kind: runtime.ErrNetwork, Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		kind := runtime.ErrNetwork
		if ctx.Err() == context.Canceled {
			kind = runtime.ErrCancellation
		} else if ctx.Err() == context.DeadlineExceeded {
			kind = runtime.ErrTimeout
		}
		return runtime.GenerationResult{}, &runtime.RuntimeError{Kind: kind, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		kind := runtime.ErrModel
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
				var r ollamaResponse
				if err := json.Unmarshal([]byte(line), &r); err != nil {
					continue
				}
				if r.Error != "" {
					ch <- runtime.StreamEvent{Err: &runtime.RuntimeError{Kind: runtime.ErrModel, Err: fmt.Errorf("%s", r.Error)}}
					return
				}
				if r.Message.Content != "" {
					ch <- runtime.StreamEvent{Chunk: r.Message.Content}
				}
				if r.Done {
					usage = runtime.Usage{PromptTokens: r.PromptEvalCount, CompletionTokens: r.EvalCount, TotalTokens: r.PromptEvalCount + r.EvalCount}
					ch <- runtime.StreamEvent{Done: true, Usage: usage}
					return
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
	// Ollama non-stream returns single JSON object.
	var r ollamaResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return runtime.GenerationResult{}, &runtime.RuntimeError{Kind: runtime.ErrModel, Err: err}
	}
	if r.Error != "" {
		return runtime.GenerationResult{}, &runtime.RuntimeError{Kind: runtime.ErrModel, Err: fmt.Errorf("%s", r.Error)}
	}
	usage := runtime.Usage{PromptTokens: r.PromptEvalCount, CompletionTokens: r.EvalCount, TotalTokens: r.PromptEvalCount + r.EvalCount}
	return runtime.GenerationResult{Text: r.Message.Content, Usage: usage}, nil
}
