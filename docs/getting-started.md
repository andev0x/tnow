# Getting started

This guide takes you from zero to a working translation using a local model, then shows how to connect a hosted provider.

## Prerequisites

- **Go 1.27+** if you build from source.
- A terminal with UTF-8 support.
- One of:
  - a local [Ollama](https://ollama.com/) installation, or
  - an API key for OpenAI, Groq, Gemini, or OpenRouter.

The default configuration uses Ollama at `http://localhost:11434` with the `llama3.2` model. If you prefer a hosted provider, skip to [Use a hosted provider](#use-a-hosted-provider).

## Install

```bash
go install github.com/andev0x/tnow/cmd/tnow@latest
```

Ensure the Go bin directory is on your `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Alternatively, build from a clone:

```bash
git clone https://github.com/andev0x/tnow.git
cd tnow
go build -o tnow ./cmd/tnow
./tnow --help
```

## Use a local model (Ollama)

Start Ollama and pull a multilingual model:

```bash
ollama serve
ollama pull llama3.2
```

Translate text:

```bash
tnow "Hello world"
```

On a fresh install, the target language is Vietnamese (`vi`). To choose another target:

```bash
tnow --to ja "Hello world"
tnow --to fr "Good morning"
```

You can also pipe input:

```bash
echo "The build passed." | tnow --to de
```

If the model is not available, `tnow` reports a provider error on stderr and exits with status `1`. Verify that `ollama serve` is running and that the model is installed:

```bash
ollama list
```

## Use a hosted provider

Set the provider's API key and select it explicitly:

```bash
export OPENAI_API_KEY="sk-..."
tnow --provider openai --model gpt-4o-mini --to vi "Hello world"
```

The same pattern works for the other providers:

```bash
export GROQ_API_KEY="..."
tnow --provider groq --model llama-3.3-70b-versatile "Hello world"

export GEMINI_API_KEY="..."
tnow --provider gemini --model gemini-2.0-flash "Hello world"

export OPENROUTER_API_KEY="..."
tnow --provider openrouter --model openai/gpt-4o-mini "Hello world"
```

API keys are loaded from the environment only when they are not already present in the config file. They are never printed.

To make a provider the default, edit the config file:

```yaml
default_provider: openai
default_model: gpt-4o-mini
target_lang: vi
```

See [Configuration](configuration.md) for file locations and the full schema.

## Analyze a word

Analysis returns IPA, part of speech, concise definitions, translations, and examples.

```bash
tnow --word awkward
tnow --word --to vi serendipity
```

The model is asked for strict JSON, which `tnow` parses and validates before rendering:

```text
awkward
────────────────────────────

/ˈɔː.kwəd/

ADJECTIVE

Meanings

• causing difficulty; hard to do or deal with
  khó xử, awkward
```

If the model returns malformed JSON or an empty definition, the request fails with a validation error instead of printing partial output.

## Open the TUI

Run `tnow` with no text and no piped stdin:

```bash
tnow
```

The TUI opens with the input pane focused.

1. Type or paste text.
2. Press `Alt+Enter` to translate.
3. Press `Tab` to switch to **Analyze** mode for word analysis.
4. Press `Ctrl+L` to cycle the target language, or type `/to ` to pick one from the palette.
5. Press `Ctrl+D` to quit.

See the [TUI guide](tui.md) for the complete key map and slash commands.

## Configure provider keys interactively

Inside the TUI:

1. Press `Ctrl+E`, or open `/models` and press `Ctrl+E`.
2. Enter the API key in the masked prompt.
3. Press `Enter`.

The key is saved to the config files with `0600` permissions, and the provider is re-initialized immediately.

## Verify your setup

```bash
# Formatting and tests (from a clone)
go test ./...

# A simple translation
tnow --to en "Xin chào"

# Word analysis
tnow --word hello
```

## Troubleshooting

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `provider unavailable` | Ollama is not running or the endpoint is wrong | Start `ollama serve`; check `OLLAMA_HOST` |
| `authentication` | Missing or invalid API key | Set the provider's environment variable or use `Ctrl+E` |
| `model error` | Model name not available to the provider | Use `--model` with an installed/authorized model |
| `validation: empty translation output` | Model returned an empty or filtered response | Retry, or choose a different model |
| `validation: conversational filler` | Model replied with prose instead of a translation | Retry with a more instruction-following model |
| TUI does not start | No TTY, or text was passed on stdin | Run `tnow` directly in a terminal |
| Clipboard copy does nothing | No clipboard utility available on the system | Install `pbcopy` (macOS), `xclip`/`wl-copy` (Linux), or disable `auto_copy` |

## Next steps

- [CLI reference](cli.md) — script-friendly usage and exit codes.
- [TUI guide](tui.md) — modes, themes, and the model browser.
- [Languages](languages.md) — accepted language names and country aliases.
- [Architecture](architecture.md) — how requests flow through the system.
