package runtime

import (
	"context"
	"errors"
	"testing"
)

type mockProvider struct {
	name string
	res  GenerationResult
	err  error
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Generate(ctx context.Context, req GenerationRequest) (GenerationResult, error) {
	if m.err != nil {
		return GenerationResult{}, m.err
	}
	return m.res, nil
}

func TestExecuteSuccess(t *testing.T) {
	mp := &mockProvider{name: "mock", res: GenerationResult{Text: "hi"}}
	rt := New(map[string]Provider{"mock": mp}, "mock")
	r, err := rt.Execute(context.Background(), GenerationRequest{Provider: "mock", Model: "m", UserPrompt: "x"})
	if err != nil || r.Text != "hi" {
		t.Fatalf("got %v %v", r, err)
	}
}

func TestExecuteUnknownProvider(t *testing.T) {
	rt := New(map[string]Provider{}, "mock")
	_, err := rt.Execute(context.Background(), GenerationRequest{Provider: "nope", Model: "m"})
	if err == nil {
		t.Fatal("expected err")
	}
}

func TestExecuteCancellation(t *testing.T) {
	mp := &mockProvider{name: "mock", err: context.Canceled}
	rt := New(map[string]Provider{"mock": mp}, "mock")
	_, err := rt.Execute(context.Background(), GenerationRequest{Provider: "mock", Model: "m"})
	if Classify(err) != ErrCancellation {
		t.Fatalf("expected cancellation, got %v", Classify(err))
	}
}

func TestExecuteTimeout(t *testing.T) {
	mp := &mockProvider{name: "mock", err: context.DeadlineExceeded}
	rt := New(map[string]Provider{"mock": mp}, "mock")
	_, err := rt.Execute(context.Background(), GenerationRequest{Provider: "mock", Model: "m"})
	if Classify(err) != ErrTimeout {
		t.Fatalf("expected timeout, got %v", Classify(err))
	}
}

func TestExecuteModelRequired(t *testing.T) {
	mp := &mockProvider{name: "mock"}
	rt := New(map[string]Provider{"mock": mp}, "mock")
	_, err := rt.Execute(context.Background(), GenerationRequest{Provider: "mock"})
	if err == nil || Classify(err) != ErrConfig {
		t.Fatalf("expected config err")
	}
	_ = errors.New
}
