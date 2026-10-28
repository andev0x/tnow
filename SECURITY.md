# Security policy

## Supported versions

`tnow` is under active development. Security fixes are applied to the latest release and the `main` branch.

| Version | Supported |
| --- | --- |
| Latest `0.x` release | Yes |
| Older releases | Best effort |
| Unreleased `main` | Yes |

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

Use GitHub's private vulnerability reporting:

1. Go to the repository's **Security** tab.
2. Select **Advisories** → **Report a vulnerability**.
3. Include a description, affected version/commit, reproduction steps, and impact.

If private reporting is unavailable, contact the maintainer through GitHub at [@andev0x](https://github.com/andev0x) and ask for a private channel. Do not include exploit details in a public message.

Please include:

- Affected version, commit, or branch
- Operating system and Go version
- The provider and model in use, if relevant
- Minimal reproduction steps or a proof of concept
- Whether secrets, local files, or unexpected network requests are involved

You can expect an acknowledgement as soon as practical. Timelines are best-effort; this is a volunteer-maintained project. Please allow time for a fix before public disclosure, and coordinate disclosure timing with the maintainer.

## Scope

In scope:

- Leakage of API keys or other secrets through output, logs, error messages, crash reports, or config permissions
- Command injection, path traversal, or unsafe file handling
- Unexpected network destinations, redirects, or credential forwarding
- Denial of service caused by malformed provider responses or streamed data
- TUI escape-sequence injection or terminal corruption from untrusted model output
- Dependency vulnerabilities that affect `tnow`

Out of scope:

- Vulnerabilities in Ollama, OpenAI, Groq, Gemini, OpenRouter, or other upstream services
- Model-generated content that is incorrect, offensive, or unsafe
- Prompt injection that only changes translation/analysis output
- Issues requiring an attacker who already controls the user's account, config files, or shell
- Rate-limit or billing issues with a provider

## Secret handling

`tnow` is designed to avoid persisting prompts and histories, but it does handle API keys:

- Config files are written with mode `0600`; the config directory is created with `0700`.
- The YAML config and JSON mirror both contain provider keys.
- Keys are loaded from environment variables when configured.
- Keys are not printed by `tnow`; provider errors include HTTP status and response bodies, not authorization headers.

Recommendations:

- Prefer environment variables or a secret manager over storing keys in config.
- Use a dedicated API key with minimal scope and spending limits.
- Do not commit `~/.config/tnow/config.yaml` or `~/.tnow/config.json` to version control.
- On shared machines, use `TNOW_CONFIG` with a private temporary file and remove it afterwards.
- Rotate a key immediately if you suspect it was exposed.

Example for isolated runs:

```bash
OPENAI_API_KEY="$SECRET" TNOW_CONFIG=/tmp/tnow-session.yaml tnow --provider openai "Hello"
rm -f /tmp/tnow-session.yaml
```

## Privacy notes

- Prompts are sent to the provider selected in configuration.
- Translation and analysis results are not stored by `tnow`.
- The TUI may copy results to the system clipboard when `auto_copy` is enabled (default).
- No telemetry is collected by `tnow`.

## Dependency security

Dependencies are declared in `go.mod` and pinned by `go.sum`. To audit locally:

```bash
go list -m all
govulncheck ./...
```

CI runs tests and linting on the current dependency set. Dependency updates are proposed through Dependabot.
