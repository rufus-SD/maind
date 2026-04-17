<div align="center">

# maind

**Your local, encrypted, persistent developer memory.**

Store decisions, bugs, solutions, and context across all your projects.
Private by design. No cloud. No accounts. Just your brain, extended.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/rufus-SD/maind/actions/workflows/ci.yml/badge.svg)](https://github.com/rufus-SD/maind/actions)

</div>

---

```
  MAIND  — krnl
  encrypted · unlocked · 42 memories · 18 tags · 7 links

  ── Activity ──
  10:32:15  STORE     decision: Use JWT with refresh tokens [a3f8c912]
  10:31:02  RECALL    "auth strategy" → 3 found
  10:28:44  SCAN_DONE entries=12 project=myapp [0f4f4f98]
  10:15:33  STORE     solution: Fixed OOM with streaming parser [b7e2d1a0]

  ──────────────────────────────────────────────
  maind>
```

## Why Maind?

You solve a gnarly bug. You make an architecture decision. You discover a pattern.
Two weeks later — gone. Buried in Slack, lost in git history, forgotten.

Maind captures that knowledge **once** and makes it available **forever** — to you and to your AI tools.

- Works across every project on your machine
- Encrypted at rest (AES-256-GCM + Argon2id)
- AI assistants read and write to it automatically
- Single binary, zero dependencies, fully offline

## Install

```bash
# From source (Go 1.21+)
go install github.com/rufus-SD/maind@latest

# Or clone and build
git clone https://github.com/rufus-SD/maind.git
cd maind && make install
```

## Get started in 60 seconds

```bash
# 1. Initialize — pick a name, enable encryption
maind init

# 2. Store something
maind remember "Always use context.WithTimeout for DB calls" \
  --kind decision --tags go,database --importance 7

# 3. Find it later
maind recall "database timeout"

# 4. Launch the dashboard
maind
```

The dashboard manages your encryption session — enter your passphrase once, work freely, it locks when you exit.

## Connect your AI

One command wires Maind into your AI assistant. It will read your memories at session start, store new ones silently, and search when it needs context.

```bash
maind connect cursor      # Cursor IDE
maind connect claude      # Claude Code
maind connect windsurf    # Windsurf
maind connect copilot     # GitHub Copilot
maind connect aider       # Aider
maind connect generic     # Any other tool — generates rules + instructions
```

Once connected, the AI treats Maind as its own memory. Decisions get stored. Bugs get logged. Context carries across sessions — without you lifting a finger.

## Deep scan

With the dashboard running and an AI connected, just tell it:

> **"scan this project"**

The AI will analyze the codebase, log its reasoning, and populate your memory with what matters — architecture decisions, dependency choices, known issues, patterns.

Every scan is a tracked session. From the dashboard:

```
maind> scan
  [0f4f4f98] DONE  04-17 10:28  myapp        12 entries  3m
         Found 8 architectural decisions, 3 tech debt items, 1 security concern

maind> scan show 0f4f
  Scan [0f4f4f98]
  Project:  myapp
  Status:   COMPLETED
  Duration: 3m12s
  Entries:  12

  Thought log:
    [10:25:23] Analyzing go.mod — 14 direct dependencies, no pinned versions
    [10:26:01] Found 3 migration files with no rollback strategy
    [10:27:15] README mentions Redis cache but no config found in deployment
    ...

  Memories created:
    [a3f8c912] decision — JWT with refresh tokens for auth
    [b7e2d1a0] bug — Migration 002 has no rollback path
    ...
```

The AI does the thinking. Maind stores the results. You review when you want.

## Commands

| Command | What it does |
|---|---|
| `maind` | Launch the live dashboard |
| `maind init` | Guided setup wizard |
| `maind remember "..."` | Store a memory |
| `maind recall "..."` | Search memories |
| `maind list` | List recent memories |
| `maind show <id>` | Full detail view |
| `maind forget <id>` | Archive (soft delete) |
| `maind link <from> <to>` | Link related memories |
| `maind tags` | List all tags |
| `maind scan start/log/complete` | Manage scan sessions |
| `maind scan list / show <id>` | Review past scans |
| `maind connect <tool>` | Wire into an AI assistant |
| `maind unlock / lock` | Manage encryption session |
| `maind export` | Export as JSON |
| `maind ingest` | Batch import (JSON lines) |
| `maind shell` | Interactive REPL |

## Memory kinds

| Kind | Use for |
|---|---|
| `note` | General knowledge |
| `decision` | Architecture / design choices |
| `bug` | Known issues, gotchas |
| `solution` | How you fixed something |
| `context` | Background info that explains why |
| `snippet` | Useful code patterns |
| `learning` | Non-obvious things you figured out |

## How encryption works

All memory content is encrypted at rest using **AES-256-GCM**. The key is derived from your passphrase with **Argon2id** (3 iterations, 64 MB, 4 threads).

```
maind               → enter passphrase → session key cached (12h, user-only)
maind remember ...  → uses cached key, no prompt
maind recall ...    → uses cached key, no prompt
exit                → session key deleted, brain locked
```

A verification token created during setup ensures wrong passphrases are rejected immediately — no silent corruption.

## Data storage

Everything lives in a single local SQLite database.

| OS | Default path |
|---|---|
| macOS | `~/Library/Application Support/maind/` |
| Linux | `~/.local/share/maind/` |
| Windows | `%APPDATA%/maind/` |

Override with `--data-dir` or choose during `maind init`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## Security

See [SECURITY.md](SECURITY.md) for the encryption model and vulnerability reporting.

## License

[MIT](LICENSE)
