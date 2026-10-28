# Configuration

`tnow` uses a YAML config file as its primary configuration and keeps a JSON mirror for compatibility with the TUI's provider/key management. Both files are created with private permissions.

## File locations

| File | Path |
| --- | --- |
| YAML (primary) | platform config directory + `tnow/config.yaml` |
| JSON (mirror) | `~/.tnow/config.json` |

Platform config directory:

| Platform | YAML path |
| --- | --- |
| macOS | `~/Library/Application Support/tnow/config.yaml` |
| Linux | `~/.config/tnow/config.yaml` (or `$XDG_CONFIG_HOME/tnow/config.yaml`) |
| Windows | `%AppData%\tnow\config.yaml` |

Override either path with environment variables:

```bash
TNOW_CONFIG=/path/to/config.yaml tnow "Hello"
TNOW_JSON_CONFIG=/path/to/config.json tnow "Hello"
```

When `TNOW_CONFIG` is set, the JSON mirror is disabled unless `TNOW_JSON_CONFIG` is also set. This is useful for isolated or throwaway runs.

## Defaults

A fresh install behaves as if this configuration were present:

```yaml
default_provider: ollama
default_model: translategemma:4b
source_lang: auto
target_lang: en

theme: tokyo-night
timeout: 60
auto_copy: true

providers:
  - name: ollama
    base_url: http://localhost:11434
    model: translategemma:4b
```

If `default_model` is empty, the model configured for the active provider is used.

## Full schema

```yaml
# Provider used when --provider is not given.
default_provider: ollama

# Model used when --model is not given. Falls back to the provider's model.
default_model: translategemma:4b

# Source language. "auto" enables heuristic detection.
source_lang: auto

# Target language. Accepts codes, names, or country aliases.
target_lang: en

# TUI theme. See "Themes" below.
theme: tokyo-night

# Optional theme aliases (checked in this order after "theme").
color_theme: ""
theme_name: ""
color_scheme: ""

# Optional per-color overrides for the selected theme.
colors:
  accent: "#7aa2f7"
  bg_base: "#1a1b26"

# Runtime timeout in seconds. Also used as the default provider timeout.
timeout: 60

# Automatically copy successful TUI results to the clipboard.
auto_copy: true

providers:
  - name: ollama
    base_url: http://localhost:11434
    api_key: ""
    model: translategemma:4b
    org: ""
    project: ""
    timeout: 0          # 0 = use the top-level timeout

  - name: openai
    base_url: https://api.openai.com/v1
    api_key: sk-...
    model: gpt-4o-mini
    org: ""             # optional OpenAI-Organization header
    project: ""         # optional project metadata
    timeout: 90
```

### Provider fields

| Field | Description |
| --- | --- |
| `name` | Provider identifier. See [Providers](#providers) |
| `base_url` | API base URL, without a trailing slash requirement |
| `api_key` | API key. Prefer environment variables or `Ctrl+E` where possible |
| `model` | Default model for this provider |
| `org` | OpenAI organization header |
| `project` | Stored project metadata (reserved for future use) |
| `timeout` | Per-provider timeout in seconds; `0` uses top-level `timeout` |

## Precedence

Configuration is resolved in this order, later values winning where they are set:

1. Built-in defaults.
2. YAML config file.
3. JSON mirror (providers, keys, defaults, theme, and color overrides are merged in).
4. Environment variables.
5. Command-line flags (`--provider`, `--model`, `--from`, `--to`).
6. TUI actions (`Ctrl+E`, model browser, `/to`, `/theme`).

Because CLI and TUI changes go through the app setters, they are saved back to the YAML file (and JSON mirror) and become the new defaults.

## Environment variables

| Variable | Effect |
| --- | --- |
| `OLLAMA_HOST` | Overrides the `ollama` provider's base URL |
| `OPENAI_API_KEY` | Adds/updates the OpenAI provider key if no key is configured |
| `GROQ_API_KEY` | Adds/updates the Groq provider key if no key is configured |
| `GEMINI_API_KEY` | Adds/updates the Gemini provider key if no key is configured |
| `OPENROUTER_API_KEY` | Adds/updates the OpenRouter provider key if no key is configured |
| `TNOW_CONFIG` | Overrides the YAML config path |
| `TNOW_JSON_CONFIG` | Overrides the JSON mirror path |

Environment-provided keys are only applied when the corresponding provider has no stored key. To rotate a key, update the config or use `Ctrl+E`; or unset the stored key first.

## Providers

The following provider names are recognized by the provider factory:

| Name | Transport | Default base URL | Requires key |
| --- | --- | --- | --- |
| `ollama` | Ollama `/api/chat` | `http://localhost:11434` | No |
| `openai` | OpenAI-compatible `/chat/completions` | `https://api.openai.com/v1` | Yes |
| `openai-compatible` | OpenAI-compatible | Defaults to the OpenAI URL if unset; configure explicitly | Provider does not require one, but the TUI prompts for it |
| `groq` | OpenAI-compatible | `https://api.groq.com/openai/v1` | Yes |
| `gemini` | OpenAI-compatible | `https://generativelanguage.googleapis.com/v1beta/openai` | Yes |
| `openrouter` | OpenAI-compatible | `https://openrouter.ai/api/v1` | Yes |

### Ollama

```yaml
default_provider: ollama
default_model: translategemma:4b
providers:
  - name: ollama
    base_url: http://localhost:11434
    model: translategemma:4b
```

```bash
ollama serve
ollama pull translategemma:4b
OLLAMA_HOST=http://192.168.1.10:11434 tnow "Hello"
```

### OpenAI

```yaml
default_provider: openai
default_model: gpt-4o-mini
providers:
  - name: openai
    base_url: https://api.openai.com/v1
    model: gpt-4o-mini
```

```bash
export OPENAI_API_KEY="sk-..."
tnow "Hello"
```

### Groq

```yaml
default_provider: groq
default_model: llama-3.3-70b-versatile
providers:
  - name: groq
    base_url: https://api.groq.com/openai/v1
    model: llama-3.3-70b-versatile
```

### Gemini

```yaml
default_provider: gemini
default_model: gemini-2.0-flash
providers:
  - name: gemini
    base_url: https://generativelanguage.googleapis.com/v1beta/openai
    model: gemini-2.0-flash
```

### OpenRouter

```yaml
default_provider: openrouter
default_model: openai/gpt-4o-mini
providers:
  - name: openrouter
    base_url: https://openrouter.ai/api/v1
    model: openai/gpt-4o-mini
```

### Self-hosted or other OpenAI-compatible endpoints

Use the `openai-compatible` provider name and set `base_url`:

```yaml
default_provider: openai-compatible
default_model: local-model
providers:
  - name: openai-compatible
    base_url: http://localhost:8000/v1
    model: local-model
    api_key: optional-token
```

The endpoint must support `POST /chat/completions`, and should support `GET /models` if you want model discovery in the browser. Model discovery failures are reported in the TUI but do not prevent translation with a configured model.

## Themes

Built-in theme names:

```text
tokyo-night
catppuccin-macchiato
dracula
nord
forest
matrix
light
dark
default
```

Set a theme in config:

```yaml
theme: dracula
```

Or in the TUI:

```text
/theme dracula
```

`color_theme`, `theme_name`, and `color_scheme` are accepted as aliases when `theme` is empty.

### Custom colors

Override individual colors on top of the selected theme:

```yaml
theme: tokyo-night
colors:
  bg_base: "#16161e"
  bg_panel: "#1a1b26"
  accent: "#7aa2f7"
  text_primary: "#c0caf5"
  success: "#9ece6a"
  danger: "#f7768e"
```

Accepted keys and aliases:

| Key | Aliases |
| --- | --- |
| `bg_base` | `bgbase`, `base` |
| `bg_panel` | `bgpanel`, `panel` |
| `bg_raised` | `bgraised`, `raised` |
| `bg_bar` | `bgbar`, `bar` |
| `bg_pill` | `bgpill`, `pill` |
| `bg_selection` | `bgselection`, `selection` |
| `bg_shadow` | `bgshadow`, `shadow` |
| `bg_card` | `bgcard`, `card` |
| `border_subtle` | `bordersubtle`, `subtle` |
| `border_focused` | `borderfocused`, `focused`, `accent` |
| `border_idle` | `borderidle`, `idle` |
| `text_primary` | `textprimary`, `primary_text`, `fg`, `foreground` |
| `text_secondary` | `textsecondary` |
| `text_muted` | `textmuted`, `muted` |
| `text_dim` | `textdim`, `dim` |
| `primary` | — |
| `secondary` | — |
| `teal` | — |
| `success` | — |
| `warning` | — |
| `danger` | `error` |
| `highlight_bg` | `highlightbg` |
| `highlight_fg` | `highlightfg` |

Keys are case-insensitive, a leading `color_` is ignored, and `-` is treated as `_`.

## API keys and permissions

- Config files are written with mode `0600`; the containing directory is created with `0700`.
- `Ctrl+E` writes the key to both the YAML file and JSON mirror.
- Keys are never printed, included in status output, or logged.
- The JSON mirror exists for compatibility; treat both files as secrets.

For CI or shared machines, prefer environment variables or a temporary config:

```bash
OPENAI_API_KEY="$SECRET" TNOW_CONFIG=/tmp/tnow-ci.yaml tnow --provider openai "Hello"
```

## JSON mirror

`~/.tnow/config.json` uses the same field names as the YAML file:

```json
{
  "default_provider": "openai",
  "default_model": "gpt-4o-mini",
  "source_lang": "auto",
  "target_lang": "en",
  "theme": "tokyo-night",
  "timeout": 60,
  "auto_copy": true,
  "providers": [
    {
      "name": "openai",
      "base_url": "https://api.openai.com/v1",
      "api_key": "sk-...",
      "model": "gpt-4o-mini"
    }
  ]
}
```

The TUI can read either file. If both exist, JSON provider entries can supply newer keys, models, or base URLs, and are merged into the YAML configuration.

## Troubleshooting configuration

| Problem | Fix |
| --- | --- |
| Config changes do not apply | Check whether `TNOW_CONFIG` points somewhere else |
| Environment key ignored | A key already exists in config; remove it or update it |
| Theme not found | Use one of the built-in names or check the `theme:` spelling |
| Provider skipped | The provider name is not recognized by the factory |
| Permission denied | Ensure the config directory is writable by your user |
