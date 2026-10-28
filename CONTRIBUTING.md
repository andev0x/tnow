# Contributing to tnow

Thanks for taking the time to contribute. `tnow` is a small, focused tool, and changes that keep it small and predictable are especially welcome.

## Ways to contribute

- Report a bug with a minimal reproduction.
- Suggest a focused improvement to translation, analysis, the TUI, or documentation.
- Add or improve provider support.
- Add language aliases or resolution tests.
- Improve documentation, examples, and error messages.
- Review pull requests.

For security issues, do **not** open a public issue. Follow [SECURITY.md](SECURITY.md).

## Before you start

- Search existing issues and pull requests to avoid duplicate work.
- For larger changes, open an issue first to agree on the approach.
- Keep pull requests focused on one concern.
- By participating, you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Development setup

### Requirements

- Go **1.27+**
- `git`
- `make` (optional)
- [golangci-lint](https://golangci-lint.run/) v2 for linting
- A provider for manual testing (Ollama is the easiest)

### Clone and build

```bash
git clone https://github.com/andev0x/tnow.git
cd tnow
go build ./...
go build -o tnow ./cmd/tnow
```

### Run the tests

```bash
go test ./...
go test -race ./...
```

### Lint and format

```bash
gofmt -w .
go vet ./...
golangci-lint run ./...
```

Or with the Makefile:

```bash
make fmt
make vet
make test
make race
make lint
```

Install golangci-lint if needed:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# or
brew install golangci-lint
golangci-lint --version   # must be v2.x
```

The repository uses [`.golangci.yml`](.golangci.yml) with the standard linter set, `go vet` checks, and `gofmt`/`goimports` (local prefix `github.com/andev0x/tnow`).

## Project layout

```text
cmd/tnow/            entrypoint
internal/app/        application wiring and model metadata
internal/analysis/   word analysis contract, parsing, validation
internal/cli/        flag parsing, stdin, CLI output
internal/config/     YAML/JSON config, env, permissions
internal/language/   language identifiers, aliases, detection
internal/provider/   Ollama and OpenAI-compatible transports
internal/runtime/    generation contracts, timeouts, error kinds
internal/translation translation contract, validation, engines
internal/tui/        Bubble Tea UI, layout, themes, browser
docs/                user and developer documentation
```

See [docs/architecture.md](docs/architecture.md) for the request lifecycle and extension points.

## Making changes

### Code style

- Run `gofmt` before committing.
- Prefer small, explicit functions over clever abstractions.
- Keep the runtime generic: translation and analysis prompts belong in their domain packages.
- Return typed `runtime.RuntimeError` values for provider/runtime failures where appropriate.
- Do not print secrets. Do not log API keys or authorization headers.
- Add comments only where the reason is not obvious from the code.
- Keep the TUI layout deterministic: derive geometry from terminal size, not content length.

### Tests

Every behaviour change should have a test when practical:

- Language changes: `internal/language/*_test.go`
- Provider behaviour: `internal/provider/*_test.go` using `httptest`
- Translation/analysis contracts: package tests
- Runtime errors/timeouts: `internal/runtime/runtime_test.go`
- CLI output/exit codes: `internal/cli/cli_test.go`
- TUI layout, palette, key handling: `internal/tui/*_test.go`

Use the existing mock provider pattern in CLI tests rather than calling a real model. Tests must not require network access or real API keys.

```bash
go test ./...
go test -race ./...
```

### Adding a provider

1. Implement `runtime.Provider` (and optionally `runtime.ModelLister`).
2. Register it in `internal/provider/provider.go`.
3. Add default base URL handling in `internal/config` and `internal/app` where relevant.
4. Add tests with `httptest` for success, auth failure, server failure, and streaming if supported.
5. Document the provider in [docs/configuration.md](docs/configuration.md).

### Adding a language

1. Add the constant and display name in `internal/language/language.go`.
2. Add it to `All()`, `regionNames`, and `extraAliases` as appropriate.
3. Add resolution tests for its code, name, and any country aliases.
4. Update the table in [docs/languages.md](docs/languages.md).

### Changing the TUI

- Verify the layout at narrow widths and short heights.
- Keep the input card and result card heights stable when content changes.
- Ensure the command palette does not move the base layout.
- Update the key tables in [docs/tui.md](docs/tui.md).
- Add or update a layout test when geometry changes.

## Commit messages

Use clear, conventional-style messages. Examples:

```text
feat(tui): add theme browser
fix(provider): classify 429 responses as rate limits
docs: document OpenAI-compatible endpoints
test(language): cover country-name resolution
refactor(runtime): centralize timeout handling
```

Keep the subject imperative and under ~72 characters. Explain the "why" in the body when it is not obvious.

## Pull requests

A good pull request:

- Has a clear title and a short description of the problem and solution.
- Is scoped to one change.
- Includes tests for behaviour changes.
- Updates documentation when user-facing behaviour changes.
- Passes `go test -race ./...`, `go vet ./...`, and `golangci-lint run ./...`.
- Does not include unrelated formatting churn.

### PR checklist

```text
[ ] Tests added or updated
[ ] go test ./... passes
[ ] go test -race ./... passes
[ ] go vet ./... passes
[ ] golangci-lint run ./... passes (v2)
[ ] Documentation updated if needed
[ ] No secrets, keys, or personal config committed
[ ] Changelog updated for user-visible changes
```

## Documentation

Documentation lives in [`docs/`](docs/). Keep it accurate and example-driven:

- [Getting started](docs/getting-started.md)
- [CLI reference](docs/cli.md)
- [TUI guide](docs/tui.md)
- [Configuration](docs/configuration.md)
- [Languages](docs/languages.md)
- [Architecture](docs/architecture.md)

If you change a flag, key binding, config field, provider default, or exit code, update the corresponding document and the README summary.

## Reporting bugs

Include:

- `tnow` version or commit SHA
- Go version and OS
- The provider and model
- The exact command or TUI steps
- Expected and actual behaviour
- Error output, with secrets removed
- Whether it reproduces with a local Ollama model

Use the bug report template when available.

## Feature requests

Describe the user problem first, then the proposed behaviour. `tnow` intentionally avoids chat history, agent loops, and broad assistant features; explain why a focused addition fits the project's scope.

## License

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE).
