# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Professional documentation set under `docs/`: getting started, CLI reference, TUI guide, configuration, languages, and architecture.
- MIT license.
- Contribution guide, code of conduct, security policy, issue templates, and pull request template.
- GitHub Actions CI for formatting, vetting, testing with the race detector, building, and linting.
- Dependabot configuration for Go modules and GitHub Actions.
- `--help` now exits with status `0`.

### Changed

- README rewritten with installation, usage, provider, configuration, and project sections.

## [0.1.0] - 2026-10-28

Initial public release.

### Added

- Terminal-native translation through Ollama, OpenAI, Groq, Gemini, OpenRouter, and OpenAI-compatible endpoints.
- Word analysis mode with strict JSON schema, parsing, and validation.
- CLI with argument or stdin input, language resolution, provider/model overrides, and composable stdout output.
- Streaming Bubble Tea TUI with fixed-geometry input/result panes.
- Slash-command palette: `/to`, `/models`, `/word`, `/theme`, `/help`, `/quit`.
- Provider/model browser with translation-optimized filtering and interactive API key setup.
- Language alias resolution across codes, names, countries, country codes, endonyms, and BCP-47 tags.
- Nine built-in themes with per-color config overrides.
- YAML configuration with JSON mirror, environment overrides, and private file permissions.
- Unit tests for language resolution, configuration, runtime errors, providers, translation, analysis, CLI, and TUI layout.

[Unreleased]: https://github.com/andev0x/tnow/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/andev0x/tnow/releases/tag/v0.1.0
