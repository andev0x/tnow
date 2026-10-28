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
tnow --to vietnam "text"
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
| `Enter` | newline inside input; accepts the highlighted suggestion when the menu is open |
| `Alt+Enter` | submit (also `Ctrl+Enter` / `Ctrl+S` / `Ctrl+J` for compatibility) |
| `Tab` | switch mode (Quick ↔ Analyze); accepts a suggestion when the menu is open |
| `Ctrl+Y` | copy result to clipboard |
| `Ctrl+L` | cycle target language |
| `Ctrl+M` / `Alt+M` | set model (modal) |
| `Ctrl+H` | highlight word/section (Esc to clear) |
| `Ctrl+E` | add or replace the active provider's API key |
| `Esc` | dismiss the command palette or overlay / cancel request / clear highlight |
| `Ctrl+C` | clear the input and result |
| `Ctrl+D` | quit |
| `↑` / `↓` / `Ctrl+P` / `Ctrl+N` | navigate suggestions |
| `Tab` / `Enter` | accept the highlighted suggestion |

### Autocomplete

The input card has a fixed height. Long drafts wrap and scroll inside it, and an
empty card shows a placeholder instead of changing size. Typing `/` opens a
floating command palette over the output area; it has a fixed size derived only
from the terminal, so opening, filtering, or closing it never resizes or moves
the input or translation panels. The palette also works when the slash command
is on the last line of a longer translation draft:

- `/` → all slash commands
- `/m` → matches `/models`
- `/to <language|country>` → set the target language; accepts codes (`vi`), language names (`vietnamese`), country names (`vietnam`, `south korea`), and country codes (`vn`, `kr`). Typing `/to ` opens the language picker and matches all of these forms.
- `/models [query]` → open the provider/model browser (aliases `/model`, `/provider`, `/providers`)
- `/word [text]` → toggle word analysis mode (Tab also toggles)
- `/quit` → exit (aliases `/exit`, `/q`)
- `/help` → show help

Navigation is keyboard-only: `↑↓` to move, `Tab`/`Enter` to accept, `Esc` to
dismiss the palette while keeping any draft above the slash line. Enter applies
a resolved target immediately; ambiguous input keeps the picker open so you can
choose. Source language remains `auto` by default and is detected from the
submitted text.

The interface uses opaque layered surfaces (terminal background, panel, raised
palette), top-left IDE-style panel titles, and a focus accent that follows the
active input or output panel. All card heights come from the terminal size only,
so the layout stays stable in split panes and short windows; long text scrolls
inside the editor and result panel, and the footer shows only shortcuts relevant
to the current state.

### Models & Providers

```bash
/models          # browse providers and models
/models gpt-4o   # pre-filter the model list
/provider openai # open with the OpenAI provider selected
```

The browser keeps providers on the left and the selected provider's models on
the right. The active model is pinned first, followed by models that are strong
at multilingual translation. The model details are intentionally compact:
usage context plus the useful capabilities (`Vision`, `Tools`). The dialog has
a fixed size; long provider/model lists scroll and end with an overflow marker
such as `… 12 more` instead of making the box grow or jump.

* **Translation-optimized filtering** — models are sorted with multilingual-translation excellence first (`✦` badge for `gpt-4o`, `claude`, `gemini`, `qwen`, `nllb`, `aya`, `mistral`, `llama3`, `command-r`, etc.). Press `Ctrl+T` in the browser to toggle *only excellent* vs *all*.
* **Secure provider setup** — select a provider on the left and press `Ctrl+E`. The masked prompt accepts the key and saves it with `0600` permissions to `~/.tnow/config.json` (and the YAML config mirror), then re-initializes the provider and returns to the same browser context. Pressing `Enter` on a model that needs a key also opens the prompt directly.

## Providers

Supported:

- Ollama
- OpenAI-compatible APIs
- Groq
- Gemini
- OpenRouter

Configure via `~/.config/tnow/config.yaml` (mirrored to `~/.tnow/config.json`, where `Ctrl+E` key updates are written) or environment variables:

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