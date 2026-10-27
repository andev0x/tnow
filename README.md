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

Type any text and press `Alt+Enter` to translate. No chat — just a bounded operation.

### Keys

| Key | Action |
|-----|--------|
| `Enter` | newline inside input |
| `Alt+Enter` | submit (also `Ctrl+Enter` / `Ctrl+S` / `Ctrl+J` for compatibility) |
| `Tab` | switch mode (Quick ↔ Analyze) |
| `Ctrl+Y` | copy result to clipboard |
| `Ctrl+L` | cycle target language |
| `Ctrl+M` / `Alt+M` | set model (modal) |
| `Ctrl+H` | highlight word/section (Esc to clear) |
| `Esc` | cancel request / dismiss overlay / clear highlight |
| `Ctrl+C` | quit |
| `↑` / `↓` / `Ctrl+P` / `Ctrl+N` | navigate suggestions |
| `Tab` / `Enter` | accept highlighted suggestion |

### Autocomplete

Typing `/` opens a stable suggestion menu (fixed height — no layout jump). The list filters as you type:

- `/` → all slash commands with inline ghost `↳ /model` (`Tab` to accept)
- `/m` → `/model`, `/to` → target language, etc.
- `/model <prefix>` → models for the active provider, filtered by prefix
- `/provider <prefix>` → available providers
- `/to <lang>` → switch target language
- `/word [text]` → toggle word analysis mode (Tab also toggles)
- `/quit` → exit (aliases `/exit`, `/q`)
- `/help` → show help

Navigation is keyboard-only, minimal chrome: `↑↓` to move, `Tab`/`Enter` to complete, `Esc` to dismiss. The dropdown is capped to 5 rows with scroll windowing and padded to a constant height so content below does not shift erratically when the filter narrows.

### Models & Providers

```bash
/model          # browse models for active provider
/model gpt-4o   # switch model
/provider       # browse providers
/provider openai
```

* **Translation-optimized filtering** — models are sorted with multilingual-translation excellence first (`✦` badge for `gpt-4o`, `claude`, `gemini`, `qwen`, `nllb`, `aya`, `mistral`, `llama3`, `command-r`, etc.). Press `Ctrl+T` inside `/model` suggestions to toggle *only excellent* vs *all*.
* **Secure provider setup** — if a provider needs an API key and none is configured, `tnow` opens a masked prompt (`•` hidden). Paste the key and press `Enter` to connect; it is saved to `~/.config/tnow/config.yaml` with `0600` permissions and the provider is re-initialized automatically. You can also paste inline: `/provider openai sk-...`.

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
make lint        # golangci-lint run ./... (uses .golangci.yml, v2, go 1.27)
make lint-fix    # auto-fix where possible
```

Lint is enforced via **golangci-lint v2** (`.golangci.yml`):

- `standard` linters (`govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused`) plus minimal `go vet` checks
- `gofmt` + `goimports` formatters (local prefix `github.com/andev0x/tnow`)
- Tests excluded from `errcheck`/`dupl` noise

Install:

```bash
# via Go (pinned version matching CI)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# or via brew
brew install golangci-lint
golangci-lint --version # >= 2.15.0
```