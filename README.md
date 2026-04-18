<div align="center">

# maind

**Your AI forgets everything when you close the chat. Maind fixes that.**

One CLI. Every decision, bug, and solution — stored, encrypted, permanent.
Your AI tools plug in and remember across sessions. You never re-explain context again.

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

## The problem

You fix a critical bug. You choose JWT over sessions. You discover a deployment gotcha.

Next week, new chat — your AI has no idea. You re-explain everything. Again.

Your knowledge is scattered across Slack threads, git commits, stale docs, and closed tabs.
None of your AI tools can access any of it.

## What Maind does

| Without Maind | With Maind |
|---|---|
| AI forgets everything per session | AI remembers across all sessions |
| Context lost in Slack/docs/git | One searchable, encrypted brain |
| Re-explain the same decisions | Decisions stored once, recalled forever |
| "Why did we choose X?" — nobody knows | `maind recall "why X"` → instant answer |
| New project, blank slate | Global memory carries across projects |

## Install

```bash
# Go (1.21+)
go install github.com/rufus-SD/maind@latest

# or clone and build
git clone https://github.com/rufus-SD/maind.git
cd maind && make install

# Python (downloads binary, no Go needed)
git clone https://github.com/rufus-SD/maind.git
cd maind && pip install -e python/
```

## Try it in 30 seconds

```bash
maind demo
```

`maind demo` creates a sample brain with real-looking memories and opens the dashboard.
No setup, no passphrase — just see what it does.

## Real setup (2 minutes)

```bash
# 1. Initialize — pick a name, enable encryption
maind init

# 2. Connect your AI
maind connect cursor      # or: claude, windsurf, copilot, aider, generic

# 3. Launch the dashboard
maind
```

That's it. Your AI now has persistent memory. It stores decisions and solutions silently as you work. Next session, it remembers.

## Deep scan — the killer feature

Tell your AI:

> **"scan this project"**

It will analyze your codebase, think out loud, and populate your memory with what matters:

```
maind> scan show 0f4f
  Scan [0f4f4f98]
  Project:  myapp
  Status:   COMPLETED
  Duration: 3m12s
  Entries:  12

  Thought log:
    [10:25:23] Analyzing go.mod — 14 direct deps, none pinned. Risky for reproducible builds.
    [10:26:01] Found 3 SQL migrations with no rollback. Recovery from bad deploys is impossible.
    [10:27:15] README mentions Redis but no config in deployment manifests.

  Memories created:
    [a3f8c912] decision — JWT with refresh tokens for auth
    [b7e2d1a0] bug — Migration 002 has no rollback path
    [c4d5e6f7] learning — Redis cache is documented but never deployed
```

The AI does the thinking. Maind stores the results. Every thought is encrypted and auditable.

## Supported AI tools

```bash
maind connect cursor      # Cursor IDE
maind connect claude      # Claude Code
maind connect windsurf    # Windsurf
maind connect copilot     # GitHub Copilot
maind connect aider       # Aider
maind connect generic     # Any other tool
```

One command. The AI reads your memories at session start, stores new ones as it works, and searches when it needs context. No config files, no API keys.

## How it works

```
You (or your AI) ──→ maind remember "..." ──→ encrypted SQLite
                                                    ↓
You (or your AI) ──→ maind recall "..."  ←── search + decrypt
```

- **Single binary** — no runtime deps, no Docker, no cloud
- **AES-256-GCM** encryption with **Argon2id** key derivation
- **FTS5** full-text search across all memories
- **Session management** — enter passphrase once, work freely, auto-locks on exit
- **Wrong passphrase rejection** — verification token prevents silent corruption

## Commands

| Command | What it does |
|---|---|
| `maind` | Launch the live dashboard |
| `maind demo` | Try it instantly with sample data |
| `maind init` | Guided setup wizard |
| `maind remember "..."` | Store a memory |
| `maind recall "..."` | Search memories |
| `maind list` | List recent memories |
| `maind show <id>` | Full detail view |
| `maind forget <id>` | Archive (soft delete) |
| `maind scan start/log/complete` | Manage AI scan sessions |
| `maind scan list / show <id>` | Review scans + AI thought logs |
| `maind connect <tool>` | Wire into an AI assistant |
| `maind status` | Check if session is ready |
| `maind unlock / lock` | Manage encryption session |
| `maind link <from> <to>` | Link related memories |
| `maind tags` | List all tags |
| `maind export` | Export as JSON |

## Memory kinds

`decision` · `solution` · `bug` · `learning` · `context` · `snippet` · `note`

## Data storage

Everything local. Single SQLite file. Your machine, your data.

| OS | Default path |
|---|---|
| macOS | `~/Library/Application Support/maind/` |
| Linux | `~/.local/share/maind/` |
| Windows | `%APPDATA%/maind/` |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md) for the encryption model and vulnerability reporting.

## License

[MIT](LICENSE)
