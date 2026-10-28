# TUI guide

Running `tnow` with no text and no piped input starts the interactive terminal UI. It is a two-pane workspace for a single bounded operation — not a chat.

```bash
tnow
```

You can pre-seed options before the TUI starts:

```bash
tnow --to ja
tnow --provider openai --model gpt-4o-mini
tnow --word
```

## Layout

```text
┌ tnow ─────────────────────────────────────────────────────────────┐
│ QUICK · AUTO → VIETNAMESE                            ollama/translategemma:4b │
├───────────────────────────────┬───────────────────────────────────┤
│ Input                         │ Result                            │
│                               │                                   │
│ Type or paste text…           │ Translation appears here.         │
│                               │                                   │
│                               │                                   │
│                               │                                   │
├───────────────────────────────┴───────────────────────────────────┤
│ Alt+Enter Translate   / Commands   Tab Mode   Ctrl+D Quit          │
└───────────────────────────────────────────────────────────────────┘
```

- The **input** card has a fixed height for the current terminal size. Long drafts wrap and scroll inside it; an empty card shows a placeholder instead of resizing.
- The **result** card scrolls independently. Overflow adds `PgUp/PgDn` to the footer.
- The header shows the current mode, language pair, provider, and model.
- The footer is context-sensitive: it only lists shortcuts that apply to the current state.
- The layout depends on terminal dimensions, not content, so it stays stable in split panes and short windows.

## Modes

| Mode | Purpose | Prompt |
| --- | --- | --- |
| **Quick** | Translate text | "Type or paste text…" |
| **Analyze** | Word/phrase analysis | "Type a word or phrase…" |

Switch modes with `Tab` or `/word`. Switching modes clears the result but keeps the draft in the input card.

`Ctrl+L` cycles Vietnamese, English, Japanese, Korean, Chinese, French, Spanish, and German. Use `/to` to select any other supported language.

## Keyboard reference

### Editing and submitting

| Key | Action |
| --- | --- |
| `Enter` | Insert a newline; accepts the highlighted suggestion when a palette is open |
| `Alt+Enter` | Submit (translate or analyze) |
| `Ctrl+Enter` | Submit, where the terminal reports it distinctly |
| `Ctrl+S` | Submit |
| `Ctrl+J` | Submit |
| `Ctrl+C` | Clear input and result; cancel an in-flight request |
| `Esc` | Close a palette or overlay; cancel a request; clear a highlight |

### Modes and targets

| Key | Action |
| --- | --- |
| `Tab` | Toggle Quick ↔ Analyze |
| `Ctrl+L` | Cycle eight common target languages |
| `Ctrl+M` / `Alt+M` | Open the model dialog |
| `Ctrl+E` | Add or replace the active provider's API key |
| `Ctrl+H` | Open the result highlight prompt |

### Result handling

| Key | Action |
| --- | --- |
| `Ctrl+Y` | Copy the result to the clipboard |
| `PgUp` / `PgDn` | Scroll the result by a page |
| `Ctrl+U` / `Alt+↑` | Scroll up one line |
| `Ctrl+↓` / `Alt+↓` | Scroll down one line |
| `Esc` | Leave output focus / clear highlight |

### Application

| Key | Action |
| --- | --- |
| `Ctrl+D` | Quit |
| `↑` / `↓` / `Ctrl+P` / `Ctrl+N` | Navigate suggestions |
| `Tab` / `Enter` | Accept the highlighted suggestion |

## Slash commands

Typing `/` opens a floating command palette over the result pane. The palette has a fixed size derived from the terminal, so opening, filtering, or closing it never moves or resizes the input and result cards.

| Command | Aliases | Usage |
| --- | --- | --- |
| `/to <language\|country>` | `/lang`, `/language`, `/target` | Set the target language |
| `/models [query]` | `/model`, `/provider`, `/providers` | Browse providers and models |
| `/word [text]` | `/analyze`, `/mode` | Toggle analysis mode |
| `/theme [name]` | `/colors`, `/colour` | Choose a color theme |
| `/help` | `/?` | Show available commands in the status line |
| `/quit` | `/exit`, `/q` | Exit |

### Autocomplete behaviour

- Typing `/` lists all commands; typing `/m` filters to commands/aliases matching that prefix.
- The palette also works when the slash command is on the last line of a multi-line draft; text above the slash line is retained.
- `↑↓` navigates, `Tab` or `Enter` accepts the highlighted item, and `Esc` dismisses the palette.
- For `/to`, typing a space opens the language picker. It matches codes (`vi`), language names (`vietnamese`), country names (`vietnam`, `south korea`), and country codes (`vn`, `kr`).
- A fully resolved `/to` value submits immediately; ambiguous input keeps the picker open so you can choose.
- An empty `/to ` picker lists all supported languages and regions.

### Examples

```text
/to vietnam
/to south korea
/to zh-tw
/models anthropic
/models gpt-4o
/word serendipity
/theme dracula
/quit
```

## Model browser

Open it with `/models` or `Ctrl+M`:

```text
/models            # all configured providers/models
/models gpt-4o     # pre-filter the model list
/provider openai   # open with OpenAI selected
```

The browser keeps providers on the left and the selected provider's models on the right.

| Key | Action |
| --- | --- |
| `↑` / `↓` / `Ctrl+P` / `Ctrl+N` | Move within the focused pane |
| `→` / `←` | Move focus between providers and models |
| Type | Filter models by search query |
| `Enter` | Select a provider or apply a model |
| `Ctrl+T` | Toggle translation-optimized filtering |
| `Ctrl+E` | Save an API key for the selected provider |
| `Ctrl+C` | Clear the search query |
| `Esc` | Close the browser |

Details:

- The active model is pinned first, followed by models that are strong at multilingual translation, then alphabetical order.
- `Ctrl+T` switches between *translation-optimized only* and *all models*. The optimized view shows a `✦ optimized` badge in the search line.
- If a provider requires an API key and none is stored, `Enter` on that provider or a model opens the masked key prompt instead of applying the selection.
- Model details show the context window and useful capabilities (`Vision`, `Tools`). Unknown models show no invented values.
- The dialog has a fixed size; long lists scroll and end with an overflow marker such as `… 12 more`.

## API keys

Press `Ctrl+E` in the main view, or in the model browser with a provider selected.

1. A masked prompt opens.
2. Paste the key and press `Enter`.
3. `Esc` cancels.

The key is trimmed, stored in the config files with `0600` permissions, and the provider instance is rebuilt immediately. Keys are never rendered or logged.

## Themes

Choose a theme with `/theme`, then `Enter` to apply:

```text
/theme
/theme dracula
/theme catppuccin-macchiato
```

Built-in themes:

| Theme | Style |
| --- | --- |
| `tokyo-night` | Default; deep blue/purple dark theme |
| `catppuccin-macchiato` | Soft pastel dark theme |
| `dracula` | High-contrast purple/green dark theme |
| `nord` | Cool blue-gray dark theme |
| `forest` | Dark green theme |
| `matrix` | Black and green terminal theme |
| `light` | Light theme |
| `dark` / `default` | Neutral dark theme |

The selected theme is saved to config. You can also set `theme:` in the config file or override individual colors with the `colors:` map. See [Configuration](configuration.md#themes).

## Highlighting

`Ctrl+H` opens a highlight prompt. Enter a word or phrase from the result to emphasize it; matches are highlighted in the result pane. Press `Esc` to clear the highlight, or `Ctrl+H` again to edit the query.

## Clipboard

- `Ctrl+Y` copies the result on demand.
- With `auto_copy: true` (the default), a successful translation or analysis is copied automatically and the status line shows `auto-copied ✓`.
- Disable automatic copying in config:

```yaml
auto_copy: false
```

Clipboard support depends on the system clipboard utility used by the Go clipboard library. On headless systems, copying may silently do nothing; the translation itself is unaffected.

## Streaming and cancellation

- Translation streams into the result pane as chunks arrive.
- While a request is running, `Esc` cancels it and leaves the input intact.
- `Ctrl+D` quits immediately and cancels an in-flight request.
- Errors are shown in the status line and do not replace the previous result.

## Mouse and terminal support

Mouse cell motion is enabled. Themes in the theme browser can be selected with the mouse; the rest of the interface is keyboard-driven. Keyboard input is always sufficient.

The TUI uses the alternate screen and restores the terminal on exit. It assumes UTF-8 and works best with true-color support; themes degrade to the terminal's palette when necessary.

## Limitations

- No conversation history is retained between runs.
- There is no chat log, agent loop, or follow-up question flow.
- The TUI always performs one bounded operation per submission.
- Text is sent to the active provider only when you submit; drafts are not transmitted as you type.
