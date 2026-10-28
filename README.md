# tnow

**A terminal-native language runtime — not a chatbot.**

`tnow` turns a bounded language intent into a bounded, validated LLM operation and prints the result with minimal ceremony. It is designed to feel like `grep`, `curl`, or `git`: fast, quiet, scriptable, and predictable.

[![CI](https://github.com/andev0x/tnow/actions/workflows/ci.yml/badge.svg)](https://github.com/andev0x/tnow/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Download](https://img.shields.io/github/v/release/andev0x/tnow?label=download&logo=github)](https://github.com/andev0x/tnow/releases/latest)

```console
$ tnow --to fr "The deployment failed unexpectedly."
> Le déploiement a échoué de manière inattendue.

$ echo "Good morning" | tnow --to ja
おはようございます
```

## Why tnow?

Most LLM tools want to be a conversation. `tnow` wants to be a command.

| Chat assistant | `tnow` |
| --- | --- |
| Conversation history | No history by default |
| Agent loops and tool plans | One bounded request, one result |
| Prose, caveats, filler | Validated output only |
| Interactive-first | Pipes, stdin, stdout, exit codes |
| Opaque prompts | Deterministic translation and analysis contracts |

## Features

- **Two focused operations** — translate text, or analyze a word with IPA, part of speech, definitions, and examples.
- **Terminal-first CLI** — read from arguments or stdin, write the result to stdout, errors to stderr.
- **Streaming TUI** — a stable, keyboard-driven interface with no chat log, slash commands, and a model browser.
- **Multiple providers** — Ollama, OpenAI, Groq, Gemini, OpenRouter, and OpenAI-compatible endpoints.
- **Language resolution** — accept codes, language names, country names, or country codes (`vi`, `vietnamese`, `vietnam`, `vn`).
- **Validation where it counts** — completed CLI results are checked for empty responses and conversational filler; analysis output must satisfy a strict schema.
- **Local by default** — the default provider is a local Ollama instance; no telemetry and no conversation history.
- **Nine built-in themes** — including Tokyo Night, Catppuccin Macchiato, Dracula, Nord, Forest, Matrix, and Light.

## Requirements

- Go **1.27** or newer to build from source.
- A provider:
  - [Ollama](https://ollama.com/) running locally (default), or
  - an API key for OpenAI, Groq, Gemini, or OpenRouter.
- A terminal with UTF-8 support. The TUI works best in a modern terminal with true color.

## Installation

### `go install` (recommended)

```bash
go install github.com/andev0x/tnow/cmd/tnow@latest
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`.

### Build from source

```bash
git clone https://github.com/andev0x/tnow.git
cd tnow
go build -o tnow ./cmd/tnow
```

### Homebrew

```bash
brew tap andev0x/tap
brew install tnow
```

Or in one command:

```bash
brew install andev0x/tap/tnow
```

### Auto install (curl)

```bash
curl -fsSL https://raw.githubusercontent.com/andev0x/tnow/main/install.sh | sh
```

[![Download](https://img.shields.io/badge/download-latest-blue?logo=github)](https://github.com/andev0x/tnow/releases/latest)

### Verify

```bash
tnow --help
```

## Quick start

### 1. Start a local model

```bash
ollama serve
ollama pull translategemma:4b
```

### 2. Translate something

```bash
tnow "Hello world"
tnow --to vi "The deployment failed unexpectedly."
tnow --from en --to ja "Ship it."
```

If no target is given, the configured default is used (`vi` on a fresh install).

### 3. Pipe text

```bash
echo "Good morning" | tnow --to vi
cat notes.txt | tnow --to fr
```

### 4. Analyze a word

```bash
tnow --word awkward
tnow --word --to vi serendipity
```

### 5. Open the TUI

```bash
tnow
```

Type, then press `Alt+Enter` (or `Ctrl+S`) to translate. Use `Tab` to switch between **Quick** translation and **Analyze** mode.

## Usage

### CLI

```text
tnow [flags] [text]
```

| Flag | Short | Description |
| --- | --- | --- |
| `--from` | `-f` | Source language (default: `auto`) |
| `--to` | `-t` | Target language |
| `--word` | `-w` | Analyze a word or short phrase |
| `--model` | `-m` | Set the model (persisted as the default) |
| `--provider` | `-p` | Set the provider (persisted as the default) |

Flags may appear before or after the text. If no text argument is provided and stdin is not a terminal, `tnow` reads stdin. With no text and no piped input, it launches the TUI.

```bash
tnow "text"
tnow --to vietnam "text"          # country name resolves to Vietnamese
tnow -to=vi -m llama3.2:3b "text"
git log -1 --pretty=%B | tnow --to en
```

Translation output is written alone to stdout, making it safe to compose:

```bash
tnow --to en "$(cat message.txt)" > message.en.txt
```

### TUI

The TUI is a two-pane workspace: input on the left, result on the right. It is intentionally not a chat interface — there is no conversation history and no agent loop.

| Key | Action |
| --- | --- |
| `Enter` | Newline in the input; accepts a highlighted suggestion |
| `Alt+Enter` / `Ctrl+S` / `Ctrl+J` / `Ctrl+Enter` | Submit |
| `Tab` | Toggle Quick ↔ Analyze; accept a suggestion when a menu is open |
| `Ctrl+Y` | Copy the result to the clipboard |
| `Ctrl+L` | Cycle eight common target languages |
| `Ctrl+M` / `Alt+M` | Set the model |
| `Ctrl+H` | Highlight a word or section in the result |
| `Ctrl+E` | Add or replace the active provider's API key |
| `Esc` | Close a menu/overlay, cancel a request, or clear a highlight |
| `Ctrl+C` | Clear input and result (cancels an in-flight request) |
| `Ctrl+D` | Quit |
| `↑` / `↓` / `Ctrl+P` / `Ctrl+N` | Navigate suggestions |
| `PgUp` / `PgDn` | Scroll the result |
| `Ctrl+U` / `Alt+↑`, `Ctrl+↓` / `Alt+↓` | Scroll the result one line |

Slash commands open a floating palette over the result pane:

| Command | Aliases | Description |
| --- | --- | --- |
| `/to <language\|country>` | `/lang`, `/language`, `/target` | Set the target language |
| `/models [query]` | `/model`, `/provider`, `/providers` | Browse providers and models |
| `/word [text]` | `/analyze`, `/mode` | Toggle word analysis mode |
| `/theme [name]` | `/colors`, `/colour` | Choose a color theme |
| `/help` | `/?` | Show available commands |
| `/quit` | `/exit`, `/q` | Exit |

Inside the model browser, `Ctrl+T` toggles between translation-optimized models and all models. `Ctrl+E` saves an API key for the selected provider.

See the [TUI guide](docs/tui.md) for layout, themes, and the full key reference.

## Providers

| Provider | `name` | Default endpoint | API key |
| --- | --- | --- | --- |
| Ollama | `ollama` | `http://localhost:11434` | Not required |
| OpenAI | `openai` | `https://api.openai.com/v1` | `OPENAI_API_KEY` |
| Groq | `groq` | `https://api.groq.com/openai/v1` | `GROQ_API_KEY` |
| Gemini | `gemini` | `https://generativelanguage.googleapis.com/v1beta/openai` | `GEMINI_API_KEY` |
| OpenRouter | `openrouter` | `https://openrouter.ai/api/v1` | `OPENROUTER_API_KEY` |
| OpenAI-compatible | `openai-compatible` | Configured `base_url` | Optional |

Configure providers in `~/.config/tnow/config.yaml` (or the platform equivalent), through environment variables, or interactively with `Ctrl+E` in the TUI. See [Configuration](docs/configuration.md).

```bash
OPENAI_API_KEY=sk-... tnow --provider openai --model gpt-4o-mini "Hello"
OLLAMA_HOST=http://127.0.0.1:11434 tnow --model qwen2.5 "Hello"
```

## Configuration at a glance

Configuration is loaded from the platform config directory:

```text
macOS    ~/Library/Application Support/tnow/config.yaml
Linux    ~/.config/tnow/config.yaml
Windows  %AppData%\tnow\config.yaml
```

A JSON mirror is written to `~/.tnow/config.json` for compatibility with the TUI's key management. Both files are created with private permissions (`0600`).

```yaml
default_provider: ollama
default_model: translategemma:4b
source_lang: auto
target_lang: vi

theme: tokyo-night
timeout: 60
auto_copy: true

providers:
  - name: ollama
    base_url: http://localhost:11434
    model: translategemma:4b
  - name: openai
    api_key: sk-...
    model: gpt-4o-mini
```

Environment variables override configuration values (API keys only fill providers that have no stored key):

| Variable | Purpose |
| --- | --- |
| `OLLAMA_HOST` | Ollama base URL |
| `OPENAI_API_KEY` | OpenAI API key |
| `GROQ_API_KEY` | Groq API key |
| `GEMINI_API_KEY` | Gemini API key |
| `OPENROUTER_API_KEY` | OpenRouter API key |
| `TNOW_CONFIG` | Override the YAML config path |
| `TNOW_JSON_CONFIG` | Override the JSON config path |

Full schema and precedence rules: [docs/configuration.md](docs/configuration.md).

## Documentation

| Document | Contents |
| --- | --- |
| [Getting started](docs/getting-started.md) | Install, configure a provider, first translation |
| [CLI reference](docs/cli.md) | Flags, stdin, output, exit codes |
| [TUI guide](docs/tui.md) | Modes, keys, slash commands, themes |
| [Configuration](docs/configuration.md) | Files, schema, environment, providers |
| [Architecture](docs/architecture.md) | Layers, packages, request lifecycle, extension points |
| [Languages](docs/languages.md) | Supported languages and resolution rules |
| [Contributing](CONTRIBUTING.md) | Development setup, tests, pull requests |
| [Security](SECURITY.md) | Reporting vulnerabilities and secret handling |
| [Changelog](CHANGELOG.md) | Release history |

## Architecture

```text
CLI / TUI / stdin
        │
        ▼
   Application
   (app, config, language)
        │
        ▼
  Language engines
  (translation, analysis)
        │
        ▼
     Runtime
  (bounded generation)
        │
        ▼
    Providers
 (Ollama / OpenAI-compatible)
        │
        ▼
      LLM
```

Every request follows the same contract: a bounded intent, a deterministic system prompt, a single provider call, and validation of completed results. There are no agent loops or hidden follow-up requests.

Read [docs/architecture.md](docs/architecture.md) for the package map and extension points.

## Development

```bash
make build       # go build ./...
make test        # go test ./...
make race        # go test -race ./...
make vet         # go vet ./...
make fmt         # gofmt -w .
make lint        # golangci-lint run ./...
make lint-fix    # golangci-lint run --fix ./...
make install     # go install ./cmd/tnow
```

Linting is enforced with **golangci-lint v2** using [`.golangci.yml`](.golangci.yml). CI runs formatting checks, `go vet`, the race-enabled test suite, a build, and linting.

See [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

## Privacy

- No conversation history is stored by default.
- Prompts are sent only to the provider you configure.
- API keys are stored in config files with `0600` permissions and are never printed or logged.
- Automatic clipboard copy is enabled by default and can be disabled with `auto_copy: false`.

## License

Released under the [MIT License](LICENSE).

Copyright (c) 2026 andev0x.

## Author

**andev0x** — [github.com/andev0x](https://github.com/andev0x)

If `tnow` is useful to you, consider starring the repository or opening an issue with feedback.
