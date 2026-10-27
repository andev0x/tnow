# tnow

`tnow` is a terminal-native language runtime, not a chatbot.

It converts a bounded language intent into a bounded, validated LLM operation and presents the result with minimal ceremony.

## Why

Most LLM tools feel like chat. `tnow` feels like `grep`, `curl`, or `git`: fast, quiet, composable.

- No chat history
- No agent loops
- No conversational filler
- Deterministic contracts
- Validated output

## Install

```bash
go build ./cmd/tnow
```

Or:

```bash
go install github.com/andev0x/tnow/cmd/tnow@latest
```

## CLI usage

```bash
tnow "Hello world"
tnow --to vi "The deployment failed unexpectedly."
echo "Good morning" | tnow --to vi
tnow --from en --to vi "text"
tnow --word awkward
tnow --model llama3.2:3b "text"
tnow --provider ollama "text"
```

Output goes to stdout; errors go to stderr.

## TUI

Run with no input:

```bash
tnow
```

Keys:

- Enter: newline
- Ctrl+Enter: submit
- Tab: switch mode (Quick/Analyze)
- Ctrl+Y: copy result
- Ctrl+L: cycle target language
- Ctrl+M / Alt+M: set model
- Ctrl+H: highlight text or section (Esc to clear)
- Esc: cancel / clear highlight
- Ctrl+C: quit

## Providers

Supported:

- Ollama
- OpenAI-compatible APIs
- Groq
- Gemini
- OpenRouter

Configure via `~/.config/tnow/config.yaml` or environment variables:

- `OLLAMA_HOST`
- `OPENAI_API_KEY`
- `GROQ_API_KEY`
- `GEMINI_API_KEY`
- `OPENROUTER_API_KEY`

## Privacy

No persistent conversation history by default. Secrets are never printed or logged.

## Architecture

```
CLI/TUI/stdin -> Application/Language Engine -> Runtime -> Provider -> LLM
```

## Development

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./...
```