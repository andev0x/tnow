package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrorKind classifies runtime/provider failures.
type ErrorKind string

const (
	ErrConfig          ErrorKind = "configuration"
	ErrProviderUnavail ErrorKind = "provider unavailable"
	ErrAuth            ErrorKind = "authentication"
	ErrNetwork         ErrorKind = "network"
	ErrTimeout         ErrorKind = "timeout"
	ErrRateLimit       ErrorKind = "rate limit"
	ErrModel           ErrorKind = "model error"
	ErrValidation      ErrorKind = "validation"
	ErrCancellation    ErrorKind = "cancellation"
	ErrInput           ErrorKind = "input"
)

// RuntimeError wraps an error with a kind.
type RuntimeError struct {
	Kind ErrorKind
	Err  error
}

func (e *RuntimeError) Error() string {
	if e.Err == nil {
		return string(e.Kind)
	}
	return fmt.Sprintf("%s: %v", e.Kind, e.Err)
}

func (e *RuntimeError) Unwrap() error { return e.Err }

func Classify(err error) ErrorKind {
	if err == nil {
		return ""
	}
	var re *RuntimeError
	if errors.As(err, &re) {
		return re.Kind
	}
	// Heuristics for context errors.
	if errors.Is(err, context.Canceled) {
		return ErrCancellation
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	return ErrModel
}

// Execute runs a generation request through the selected provider.
func (r *Runtime) Execute(ctx context.Context, req GenerationRequest) (GenerationResult, error) {
	if req.Provider == "" {
		req.Provider = r.Default
	}
	p, ok := r.Providers[req.Provider]
	if !ok {
		return GenerationResult{}, &RuntimeError{Kind: ErrConfig, Err: fmt.Errorf("unknown provider %q", req.Provider)}
	}
	if req.Model == "" {
		return GenerationResult{}, &RuntimeError{Kind: ErrConfig, Err: errors.New("model is required")}
	}
	// Apply timeout.
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 60
	}
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	}
	res, err := p.Generate(ctx, req)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		if errors.Is(err, context.Canceled) {
			return GenerationResult{}, &RuntimeError{Kind: ErrCancellation, Err: err}
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return GenerationResult{}, &RuntimeError{Kind: ErrTimeout, Err: err}
		}
		return GenerationResult{}, err
	}
	res.Cancel = cancel
	return res, nil
}
