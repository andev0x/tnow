# CLI reference

`tnow` has a small, composable command-line surface. Translation text goes to stdout; diagnostics go to stderr. When no text is supplied and stdin is not piped, the TUI starts instead.

## Synopsis

```text
tnow [flags] [text...]
```

Flags may appear before or after positional text.

## Flags

| Flag | Short | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--from` | `-f` | string | config (`auto`) | Source language |
| `--to` | `-t` | string | config (`en`) | Target language |
| `--word` | `-w` | bool | `false` | Analyze a word or short phrase |
| `--model` | `-m` | string | config/provider | Model override for this run |
| `--provider` | `-p` | string | config (`ollama`) | Provider override for this run |
| `--verbose` | — | bool | `false` | Reserved; currently has no effect |
| `--debug` | — | bool | `false` | Reserved; currently has no effect |
| `--help` | — | bool | — | Print usage and exit |

`--help` is provided by Go's flag package and prints to stderr, matching standard Go CLI behaviour.

## Text input

Text is resolved in this order:

1. All positional arguments, joined with single spaces.
2. If no positional text is present and stdin is piped, stdin is read to EOF.
3. If the resulting text is empty, the TUI starts.

```bash
tnow Hello world                      # text: "Hello world"
tnow --to vi "Hello world"            # flags first
tnow "Hello world" --to vi            # flags last
echo "Hello world" | tnow --to vi     # stdin
printf 'line one\nline two\n' | tnow  # multi-line input
```

Blank input after trimming launches the TUI; it is not sent to a provider.

## Language arguments

Any value accepted by [`language.Resolve`](languages.md) can be used with `--from` and `--to`. This includes:

- ISO-like codes: `en`, `vi`, `zh-tw`, `yue`
- Language names: `english`, `vietnamese`, `cantonese`
- Country names: `vietnam`, `japan`, `south korea`
- Country codes: `vn`, `jp`, `kr`
- A small set of endonyms and Vietnamese aliases: `tiếng việt`, `日本語`, `pháp`

Unknown values are normalized to lowercase and passed through, allowing provider-specific language identifiers.

```bash
tnow --to vietnam "Hello"
tnow --to "south korea" "Hello"
tnow -f en -t ja "Hello"
```

The source language defaults to `auto`. A lightweight heuristic detects the script for metadata, but the model is instructed to translate the given text into the target language.

## Modes

### Translation (default)

```bash
tnow "Hello world"
tnow --to vi "The deployment failed unexpectedly."
```

The model is instructed to return only the translation, preserving meaning, tone, formatting, and line breaks. The response is trimmed and rejected if it is empty or begins with conversational filler such as `Sure!` or `Here is the translation`.

### Word analysis

```bash
tnow --word awkward
tnow -w --to vi serendipity
```

The model must return strict JSON describing the word. `tnow` parses it, validates required fields, and renders a compact human-readable report:

```text
awkward
────────────────────────────

/ˈɔː.kwəd/

ADJECTIVE

Meanings

• causing difficulty; hard to deal with
  khó xử

Examples

It was an awkward silence.
→ Đó là một khoảng lặng khó xử.
```

There is no JSON output flag; analysis is formatted for terminal reading.

## Provider and model overrides

```bash
tnow --provider openai --model gpt-4o-mini "Hello"
tnow -p groq -m llama-3.3-70b-versatile "Hello"
tnow --model llama3.2:3b "Hello"
```

Overrides are applied to the running app and persisted through the same setters used by the TUI. A CLI provider or model choice becomes the new default in the config file. To experiment without changing defaults, use a temporary `TNOW_CONFIG` path:

```bash
TNOW_CONFIG=/tmp/tnow-scratch.yaml tnow --provider openai --model gpt-4o-mini "Hello"
```

## Output contract

| Stream | Content |
| --- | --- |
| stdout | Translation text, or the rendered analysis report |
| stderr | Usage, configuration errors, provider errors, validation errors |

A successful translation prints exactly one line break after the result. This makes composition straightforward:

```bash
tnow --to en "$(cat message.vi.txt)" > message.en.txt
translated="$(tnow --to en "Xin chào")"
```

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success; also returned for `--help` |
| `1` | Configuration, provider, runtime, or validation error |
| `2` | Invalid flag or argument parsing error |

```bash
if tnow --to vi "$text" > out.txt; then
  echo "translated"
else
  echo "translation failed" >&2
fi
```

## Examples

### Translate a commit message

```bash
git log -1 --pretty=%B | tnow --to en
```

### Translate a file, line breaks preserved

```bash
tnow --to fr < README.md
```

### Batch a list of phrases

```bash
while IFS= read -r line; do
  printf '%s\t' "$line"
  tnow --to vi "$line"
done < phrases.txt
```

### Use a fast local model

```bash
tnow --provider ollama --model qwen2.5:3b "Hello"
```

## Interaction with configuration

The CLI loads configuration on startup. Command-line provider, model, source, and target values take precedence for that run and are written back to the config file through the app setters. Use a temporary `TNOW_CONFIG` (and optionally `TNOW_JSON_CONFIG`) if you do not want a one-off command to change your defaults.

Environment variables such as `OPENAI_API_KEY` and `OLLAMA_HOST` are applied during configuration loading. See [Configuration](configuration.md).

## Limitations

- The CLI does not stream tokens; it prints the completed result.
- `--verbose` and `--debug` are accepted but do not currently change output.
- Analysis output is human-readable only, not machine-readable JSON.
- Source-language detection is heuristic and primarily script-based; exotic or short Latin-script inputs may be labelled `en`.
