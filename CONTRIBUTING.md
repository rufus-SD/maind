# Contributing to Maind

Thanks for your interest in contributing.

## Getting started

```bash
git clone https://github.com/rufus-SD/maind.git
cd maind
make build
make test
```

Requires **Go 1.21+**. No other dependencies — SQLite is embedded via `modernc.org/sqlite`.

## Development workflow

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Add or update tests as needed
4. Run `make test` and `make lint` — both must pass
5. Open a pull request with a clear description

## Code style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions focused and small
- Error messages should be lowercase, no trailing punctuation
- CLI output goes to `stderr` (user-facing messages), `stdout` (machine-parseable data like IDs)

## Project structure

```
internal/
  cli/       Command implementations (Cobra)
  config/    Configuration loading/saving
  crypto/    Encryption (AES-256-GCM) and key derivation (Argon2id)
  model/     Data types (Entry, Tag, Link, enums)
  store/     SQLite database, migrations, queries
  tui/       Terminal UI dashboard (Bubbletea)
```

## Testing

```bash
make test        # run all tests
make lint        # go vet
```

Tests use `t.TempDir()` for isolated databases — no cleanup needed.

## What to contribute

- Bug fixes
- New memory kinds or link relations
- Search improvements
- Additional AI integrations (`maind connect <tool>`)
- Platform-specific fixes (Windows, Linux)
- Documentation and examples

## What to avoid

- Adding network/cloud dependencies — Maind is local-first by design
- Changing the encryption scheme without discussion
- Large refactors without an issue/discussion first
