# Security Policy

## Encryption model

Maind encrypts all memory content at rest using **AES-256-GCM** with keys derived via **Argon2id** (3 iterations, 64 MB memory, 4 threads, 32-byte key).

- The passphrase never leaves your machine
- The derived key is cached in a session file (`0600` permissions, 12-hour expiry)
- A verification token prevents silent encryption with a wrong passphrase
- The SQLite database is local-only — no network calls, no telemetry, no cloud sync

## Reporting a vulnerability

If you discover a security issue, please report it responsibly:

1. **Do not** open a public GitHub issue
2. Email **[TODO: add contact email]** with details
3. Include steps to reproduce if possible

You will receive a response within 48 hours. We will coordinate a fix and disclosure timeline with you.

## Scope

Security reports are welcome for:

- Encryption implementation flaws
- Key derivation weaknesses
- Session key file exposure risks
- SQL injection in query construction
- Any path where data could leak in plaintext

Out of scope:

- Attacks requiring physical access to an unlocked machine
- Social engineering
- Denial of service against the local CLI
