# Architecture

`tnow` is organized around a simple rule: a user intent becomes one bounded generation request with a deterministic prompt, and the response is validated before it is shown. There are no agent loops, conversation state, or hidden follow-up calls.

## System overview

```text
┌─────────────────────────────────────────────────────────────┐
│ cmd/tnow                                                    │
│   config.Load → app.New → cli.Run → (TUI if no input)       │
└───────────────────────────┬─────────────────────────────────┘
                            │
        ┌───────────────────┴───────────────────┐
        │                                       │
┌───────▼────────┐                     ┌────────▼────────┐
│ internal/cli   │                     │ internal/tui    │
│ flags, stdin   │                     │ Bubble Tea UI   │
└───────┬────────┘                     └────────┬────────┘
        │                                       │
        └───────────────────┬───────────────────┘
                            │
                    ┌───────▼────────┐
                    │ internal/app   │
                    │ wiring/config  │
                    └───────┬────────┘
                            │
          ┌─────────────────┴─────────────────┐
          │                                   │
 ┌────────▼─────────┐                ┌────────▼─────────┐
 │ translation      │                │ analysis         │
 │ engine+contract  │                │ engine+schema    │
 └────────┬─────────┘                └────────┬─────────┘
          │                                   │
          └─────────────────┬─────────────────┘
                            │
                    ┌───────▼────────┐
                    │ runtime        │
                    │ bounded call   │
                    └───────┬────────┘
                            │
                    ┌───────▼────────┐
                    │ providers      │
                    │ Ollama/OpenAI  │
                    └───────┬────────┘
                            │
                            ▼
                           LLM
```

## Package map

| Package | Responsibility |
| --- | --- |
| `cmd/tnow` | Process entrypoint: load config, build the app, detect piped stdin, run CLI or TUI, exit with the right code |
| `internal/cli` | Flag parsing, argument reordering, language resolution, stdin handling, analysis rendering |
| `internal/app` | Application wiring: providers, runtime, engines, settings, model metadata, search helpers |
| `internal/config` | YAML/JSON loading and saving, defaults, environment overrides, private permissions |
| `internal/language` | Language identifiers, display names, aliases, resolution, heuristic detection |
| `internal/runtime` | Generation contracts, provider interface, timeout handling, error classification |
| `internal/provider` | Ollama and OpenAI-compatible HTTP transports, streaming, model listing |
| `internal/translation` | Translation request/result, system prompt, validation, streaming engine |
| `internal/analysis` | Word analysis JSON schema, parsing, validation, engine |
| `internal/tui` | Bubble Tea model, layout, themes, model browser, highlighting, clipboard |

## Request lifecycle

### Translation (CLI)

1. `config.Load` builds a config from defaults, YAML, the JSON mirror, and environment variables.
2. `app.New` constructs provider instances, a runtime, and translation/analysis engines.
3. `cli.Run` reorders flags, parses them, applies provider/model/language overrides, and collects text from arguments or stdin.
4. `translation.Engine.Translate` resolves the source language for metadata and calls `BuildRequest`.
5. `BuildRequest` creates a `runtime.GenerationRequest` with:
   - the target language in a deterministic system prompt,
   - the user text (plus optional context),
   - `max_tokens = 2048`, `temperature = 0.2`,
   - `Operation = "translation"`.
6. `runtime.Execute` selects the provider, requires a model, and applies a timeout if the context has no deadline.
7. The provider performs one HTTP request and returns text, usage, and optional stream.
8. `translation.Validate` trims the result and rejects empty output or conversational filler.
9. The CLI writes the validated text to stdout.

### Translation (TUI)

The TUI uses `TranslateStream`, which calls the provider with `Stream: true` and forwards chunks through `tea.Cmd` messages. Cancellation is propagated through the request context. Streamed chunks are displayed as they arrive; the completed stream is not re-validated, so the prompt contract is the primary guard in TUI translation. Analysis always uses the validated, non-streaming path.

### Word analysis

1. `analysis.Engine.Analyze` resolves the source language (using heuristic detection when it is `auto`) and builds a strict JSON schema prompt that names both the input and output languages.
2. The runtime performs one non-streaming call with `max_tokens = 1024`, `temperature = 0.2`.
3. `analysis.Parse` strips accidental Markdown fences and unmarshals the JSON.
4. `analysis.Validate` requires a word and at least one definition with a meaning; examples, if present, must have source and translation.
5. Missing word fields are filled from the request; the resolved source language overwrites any model-supplied language label so the report cannot contradict the request.
6. The CLI renders a text report; the TUI renders a structured result.

## Core contracts

```go
// Provider is the inference transport boundary.
type Provider interface {
    Name() string
    Generate(ctx context.Context, req GenerationRequest) (GenerationResult, error)
}

// Optional: providers that can enumerate models.
type ModelLister interface {
    ListModels(ctx context.Context) ([]string, error)
}
```

`runtime.GenerationRequest` is deliberately small: provider, model, system prompt, user prompt, token limit, temperature, stream flag, and an operation label. Domain packages own their prompts; the runtime knows nothing about translation or analysis.

`runtime.GenerationResult` carries text, token usage, an optional stream channel, and a cancel function.

## Error model

Errors are wrapped in `runtime.RuntimeError` with a kind:

| Kind | Meaning | Typical source |
| --- | --- | --- |
| `configuration` | Missing/unknown provider, missing model | Runtime, provider factory |
| `provider unavailable` | Upstream 5xx or unreachable Ollama | HTTP status mapping |
| `authentication` | 401/403 | HTTP status mapping |
| `network` | DNS, connection, read failure | HTTP client |
| `timeout` | Deadline exceeded | Context, HTTP client |
| `rate limit` | 429 | HTTP status mapping |
| `model error` | Model rejected request, bad response, no choices | Provider response |
| `validation` | Empty output, filler, invalid JSON, missing fields | Translation/analysis validation |
| `cancellation` | User cancelled or context cancelled | Context |
| `input` | Reserved for invalid user input | — |

`runtime.Classify` returns the kind for a wrapped error, or infers cancellation/timeout from the context. The CLI prints `err.Error()` to stderr and exits `1`; the TUI displays the message in the status line.

## Design decisions

### Bounded operations over conversation

Each submission is one request. There is no message history, no tool loop, and no persistence of prompts or outputs. This keeps behaviour predictable and makes the tool composable with shell pipelines.

### Prompts live with their domain

Translation prompts belong to `internal/translation`; analysis schemas belong to `internal/analysis`. The runtime remains a generic transport boundary.

### Validate at the boundary

Model output is untrusted. Translation rejects empty responses and common conversational prefixes in the non-streaming path used by the CLI. Analysis requires strict JSON, strips accidental fences, and checks required fields. Validation failures are surfaced as errors, not silently rendered. The TUI streams translation chunks directly; it does not apply a final post-stream validation pass.

### Deterministic UI geometry

The TUI derives card and dialog sizes from the terminal dimensions only. Content changes do not resize panels; long text scrolls, and the model browser uses overflow markers. This avoids layout jumps in split panes and short windows.

### Secrets stay out of output

API keys are loaded into provider config, written with `0600` permissions, and never printed. Error messages contain HTTP status and response bodies, not authorization headers.

### Local-first defaults

The default provider is Ollama with no API key. Hosted providers are opt-in through configuration or flags.

## Extension points

### Add a provider

1. Implement `runtime.Provider` in `internal/provider`.
2. Optionally implement `runtime.ModelLister` for the model browser.
3. Register the name in `provider.New`.
4. Add default base URL handling where appropriate (`config.applyEnv`, `app.SetProvider`, `config.SetAPIKey`).
5. Add HTTP mapping tests for auth, rate limit, server errors, and streaming.

### Add a language

1. Add a `Language` constant in `internal/language/language.go`.
2. Add it to `displayNames`, `All()`, `regionNames`, and `extraAliases` as needed.
3. Add resolution tests in `internal/language/resolve_test.go`.

### Add an operation

1. Create a package with request/result types, a contract (prompt), an engine, and validation.
2. Wire an engine instance into `app.App`.
3. Add a CLI path and/or TUI mode.
4. Use `runtime.GenerationRequest.Operation` to label the call.
5. Cover success, provider failure, and invalid output paths.

### Add a theme

Register a `Theme` in `themeRegistry` in `internal/tui/theme.go`. The theme browser, `/theme` command, and config overrides pick it up automatically.

### Improve model metadata

`app.ModelInfoFor` maps model names to context windows and capabilities. Add family rules there; unknown models intentionally return no invented values.

## Testing

- Table-driven unit tests cover language resolution, config loading/permissions, translation validation, analysis parsing, and runtime error handling.
- Provider tests use `httptest` servers to exercise the Ollama and OpenAI-compatible generation paths.
- CLI tests use a mock provider and assert stdout/stderr/exit codes.
- TUI tests assert layout stability, fixed dialog geometry, slash-command behaviour, and highlighting.
- `go test -race ./...` is part of CI.

## Non-goals

- Persistent chat history or session memory.
- Autonomous agents, planning, or tool execution.
- JSON/machine-readable analysis output.
- Streaming output in CLI mode.
- Full language identification beyond script heuristics.

## Related reading

- [Configuration](configuration.md)
- [CLI reference](cli.md)
- [TUI guide](tui.md)
- [Languages](languages.md)
