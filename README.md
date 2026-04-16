# Maind

**Your second brain — persistent developer memory.**

Maind is a local CLI tool that acts as a global, persistent memory for developers. It stores decisions, bugs, solutions, context, and learnings across all your projects and sessions. Your data stays local, encrypted, and private.

## Features

- **Persistent memory** — store and recall knowledge across projects and sessions
- **Encryption at rest** — AES-256-GCM with Argon2id key derivation; passphrase verified on every unlock
- **Full-text search** — FTS5-powered search across titles, bodies, and tags
- **Live dashboard** — TUI with real-time activity monitoring and session management
- **AI integration** — connect to Cursor, Claude Code, or any AI assistant so they remember too
- **Single binary** — no runtime dependencies, no cloud, no external database
- **Linked knowledge** — connect related memories with typed relationships

## Install

### From source (requires Go 1.21+)

```bash
go install github.com/rufus-SD/maind@latest
```

### Build from repo

```bash
git clone https://github.com/rufus-SD/maind.git
cd maind
make install
```

## Quick start

```bash
# Initialize your brain (guided setup)
maind init

# Store a memory
maind remember "Use context.WithTimeout for all DB calls" --kind decision --tags go,database --importance 7

# Search your memories
maind recall "database timeout"

# List recent memories
maind list

# Start the live dashboard (manages encryption session)
maind
```

## Encryption

Maind encrypts all memory content at rest using **AES-256-GCM**. The encryption key is derived from your passphrase using **Argon2id** (3 iterations, 64 MB memory, 4 threads).

### Session model

When you start the dashboard (`maind`), you enter your passphrase once. A session key is cached locally (readable only by your user, expires in 12 hours). While the session is active, all commands work without re-prompting:

```
maind              # start dashboard, enter passphrase → session unlocked
maind remember ... # uses session key, no prompt
maind recall ...   # uses session key, no prompt
exit               # dashboard exits → session key deleted, locked
```

You can also manage sessions manually:

```bash
maind unlock       # enter passphrase, cache session key
maind lock         # delete session key
```

### Passphrase verification

Maind stores an encrypted verification token during setup. On every unlock, it checks the token — a wrong passphrase is rejected immediately instead of silently encrypting data with the wrong key.

## AI integration

Maind can integrate with AI coding assistants so they use your memory automatically.

```bash
# In your project directory:
maind connect cursor    # generates .cursor/rules/maind.mdc
maind connect claude    # appends to CLAUDE.md
```

With the dashboard running, your AI assistant will:
- Read recent memories at session start
- Store decisions, solutions, and learnings silently
- Search your memory when relevant context is needed

Allowlist `maind` once when prompted, then it's seamless.

## Commands

| Command | Description |
|---------|-------------|
| `maind` | Start the live dashboard (TUI) |
| `maind init` | Guided setup for your developer memory |
| `maind remember [text]` | Store a new memory |
| `maind recall [query]` | Search your memories |
| `maind list` | List stored memories |
| `maind show [id]` | Show full details of a memory |
| `maind forget [id]` | Archive a memory (soft delete) |
| `maind link [from] [to]` | Create a link between memories |
| `maind tags` | List all tags with entry counts |
| `maind export` | Export all memories as JSON |
| `maind connect [tool]` | Wire Maind into AI tools (cursor, claude) |
| `maind unlock` | Unlock encryption for this session |
| `maind lock` | Lock encryption, clear session key |
| `maind shell` | Start an interactive REPL session |
| `maind ingest` | Batch import from stdin (JSON lines) |
| `maind version` | Print version |

### Memory kinds

`note`, `decision`, `bug`, `solution`, `context`, `snippet`, `learning`

### Link relations

`relates_to`, `caused_by`, `supersedes`, `solved_by`, `depends_on`, `part_of`, `derived_from`

## Data storage

All data is stored locally in a single SQLite database. Default locations:

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/maind/` |
| Linux | `~/.local/share/maind/` (or `$XDG_DATA_HOME/maind/`) |
| Windows | `%APPDATA%/maind/` |

Override with `--data-dir` or choose during `maind init`.

## License

[MIT](LICENSE)
