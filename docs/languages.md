# Languages

`tnow` treats languages as short identifiers and resolves friendly input — language names, country names, country codes, endonyms, and BCP-47 tags — into a supported code before sending the request.

## Supported languages

| Code | Language | Primary region |
| --- | --- | --- |
| `en` | English | United States / international |
| `es` | Spanish | Spain |
| `fr` | French | France |
| `de` | German | Germany |
| `zh` | Chinese (Simplified) | China |
| `zh-tw` | Chinese (Traditional) | Taiwan |
| `ja` | Japanese | Japan |
| `ko` | Korean | South Korea |
| `pt` | Portuguese | Portugal / Brazil |
| `ru` | Russian | Russia |
| `vi` | Vietnamese | Vietnam |
| `hi` | Hindi | India |
| `ar` | Arabic | Saudi Arabia / Middle East |
| `th` | Thai | Thailand |
| `yue` | Cantonese | Hong Kong / Macau |
| `id` | Indonesian | Indonesia |
| `ms` | Malay | Malaysia |
| `my` | Burmese | Myanmar |
| `km` | Khmer | Cambodia |
| `lo` | Lao | Laos |
| `it` | Italian | Italy |
| `nl` | Dutch | Netherlands / Belgium |
| `cs` | Czech | Czechia |
| `hr` | Croatian | Croatia |
| `hu` | Hungarian | Hungary |
| `is` | Icelandic | Iceland |
| `ga` | Irish | Ireland |
| `eu` | Basque | Basque Country |
| `gl` | Galician | Galicia |
| `sw` | Swahili | Tanzania / East Africa |
| `ht` | Haitian Creole | Haiti |
| `uz` | Uzbek | Uzbekistan |
| `kk` | Kazakh | Kazakhstan |
| `ky` | Kyrgyz | Kyrgyzstan |

`auto` is accepted as the source language.

The model ultimately determines translation quality. Languages with less training data may need a larger or translation-specialized model.

## Resolution rules

Given a value, `tnow` tries to resolve it in this order:

1. **Exact alias match** (case-insensitive).
2. **Unique prefix match**.
3. **Unique substring match** for queries of at least three characters.
4. **BCP-47 primary subtag match** (`en-US` → `en`, `pt-BR` → `pt`, `vi-VN` → `vi`).
5. Otherwise the value is normalized to lowercase and passed through as a custom language code.

Ambiguous input is not guessed. In the TUI, `/to` leaves the picker open when several languages match; in scripts, pass an unambiguous code.

Normalization details:

- Case-insensitive: `VI`, `Vi`, `vi` are equivalent.
- Surrounding whitespace and quotes are trimmed.
- Hyphens, underscores, spaces, dots, commas, apostrophes, and backticks are treated as separators.
- Accents are preserved in aliases, so both `tiếng việt` and `tieng viet` work for Vietnamese.

## Examples

All of these resolve to Vietnamese (`vi`):

```text
vi
vie
vietnamese
tiếng việt
tieng viet
vietnam
viet nam
vn
```

Other examples:

| Input | Resolves to |
| --- | --- |
| `english`, `eng`, `usa`, `uk`, `en-GB` | `en` |
| `japanese`, `日本語`, `japan`, `jp` | `ja` |
| `korean`, `한국어`, `south korea`, `kr` | `ko` |
| `traditional chinese`, `taiwan`, `繁體中文` | `zh-tw` |
| `cantonese`, `hong kong`, `粵語` | `yue` |
| `portuguese`, `brazil`, `pt-BR` | `pt` |
| `spanish`, `mexico`, `es-419` | `es` |
| `german`, `deutsch`, `austria` | `de` |
| `french`, `français`, `senegal` | `fr` |
| `arabic`, `egypt`, `uae` | `ar` |
| `hindi`, `india` | `hi` |
| `swahili`, `kenya` | `sw` |

## Source-language detection

When the source language is `auto`, `tnow` performs a lightweight, deterministic script check:

| Signal | Detected language |
| --- | --- |
| Hiragana / katakana | Japanese |
| Hangul | Korean |
| CJK characters only | Chinese (~0.7 confidence) |
| Arabic script | Arabic |
| Devanagari | Hindi |
| Thai script | Thai |
| Khmer script | Khmer |
| Lao script | Lao |
| Myanmar script | Burmese |
| Cyrillic | Russian |
| Vietnamese diacritics in Latin text | Vietnamese |
| Other Latin text | English (~0.5 confidence) |

This detection is intentionally cheap and is not a full language classifier. It does not distinguish regional variants or closely related Latin-script languages, and short inputs may be ambiguous. It is used for request metadata; the model is still instructed to translate the provided text into the target language.

To avoid ambiguity, set the source explicitly:

```bash
tnow --from es --to en "Hola mundo"
```

## Target-language defaults

- Fresh config: `vi` (Vietnamese).
- Analysis mode: falls back to `vi` if no target is configured.
- The TUI cycles through eight common targets with `Ctrl+L` and can set any supported language through `/to`.

## Custom language codes

If a value is not recognized, `tnow` normalizes it and sends it as-is. This allows provider-specific identifiers, but it also means a typo is not rejected. Use `--to` with a known code or name when reliability matters.

```bash
tnow --to "pt-BR" "Hello"   # resolves to pt
tnow --to xx "Hello"        # passes "xx" through to the provider
```
