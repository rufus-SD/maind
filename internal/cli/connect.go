package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect [tool]",
	Short: "Wire Maind into your AI tools",
	Long: `Generate integration rules so your AI assistant uses Maind as persistent memory.

Supported tools:
  cursor    — creates .cursor/rules/maind.mdc in the current project
  claude    — creates CLAUDE.md in the current project

Examples:
  maind connect cursor
  maind connect claude`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"cursor", "claude"},
	RunE:      runConnect,
}

func runConnect(cmd *cobra.Command, args []string) error {
	tool := args[0]

	switch tool {
	case "cursor":
		return connectCursor()
	case "claude":
		return connectClaude()
	default:
		return fmt.Errorf("unknown tool %q — supported: cursor, claude", tool)
	}
}

func connectCursor() error {
	if err := os.MkdirAll(".cursor/rules", 0755); err != nil {
		return fmt.Errorf("create rules directory: %w", err)
	}
	if err := os.MkdirAll(".maind", 0755); err != nil {
		return fmt.Errorf("create .maind directory: %w", err)
	}

	os.WriteFile(filepath.Join(".cursor", "rules", "maind.mdc"), []byte(cursorRule), 0644)

	s, err := openStore()
	if err == nil {
		cwd, _ := os.Getwd()
		refreshContextFile(s, cwd)
		s.Close()
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Maind connected to Cursor.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "    Rules:   .cursor/rules/maind.mdc\n")
	fmt.Fprintf(os.Stderr, "    Context: .maind/context.md (refreshed by dashboard)\n")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  The AI will:")
	fmt.Fprintln(os.Stderr, "    - Read .maind/context.md at session start")
	fmt.Fprintln(os.Stderr, "    - Run 'maind remember' to store memories directly")
	fmt.Fprintln(os.Stderr, "    - Run 'maind recall' to search when needed")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Allowlist 'maind' once when prompted, then it's seamless.")
	fmt.Fprintln(os.Stderr)
	return nil
}

func connectClaude() error {
	path := "CLAUDE.md"

	existing := ""
	if data, err := os.ReadFile(path); err == nil {
		existing = string(data)
	}

	content := existing
	if content != "" && content[len(content)-1] != '\n' {
		content += "\n"
	}
	content += claudeRule

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write CLAUDE.md: %w", err)
	}

	abs, _ := filepath.Abs(path)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Maind connected to Claude Code.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "    File: %s\n", abs)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Claude will now use Maind as persistent memory.")
	fmt.Fprintln(os.Stderr)
	return nil
}

const cursorRule = `---
description: "Maind — persistent developer memory"
globs: "**/*"
alwaysApply: true
---

# Maind — Your Persistent Memory

You have persistent memory via Maind. It stores decisions, bugs, solutions,
context, and learnings across all sessions. Treat it as YOUR memory.

## On session start — read your memories

Your FIRST action must be to read this file with the Read tool:

    .maind/context.md

This contains your recent memories from past sessions. Read it before anything else.

## Storing memories — use the CLI directly

When you need to store a memory, run:

` + "```bash" + `
maind remember "description" --kind <kind> --tags tag1,tag2 --importance <1-10> --source ide
` + "```" + `

Valid kinds: note, decision, bug, solution, context, snippet, learning
The dashboard must be running (it manages the session key — no passphrase needed).

## When to store (do this silently, never announce it)

- A decision is made: --kind decision
- A bug is found and solved: --kind solution
- Something non-obvious is learned: --kind learning
- Important context is established: --kind context
- A useful code pattern is created: --kind snippet

Do NOT store trivial things like "ran npm install" or "read a file".
Do NOT announce or mention that you are storing a memory. Just do it silently.

## Searching memories

` + "```bash" + `
maind recall "search query"
` + "```" + `

## Importance scale

1-3: trivial, 4-6: useful, 7-8: important, 9-10: critical

## Tags

Use specific, searchable tags: "auth,jwt,refresh-tokens" not just "backend".
Use --project to scope memories to the current project when relevant.
`

const claudeRule = `
## Maind — Persistent Developer Memory

You have access to ` + "`maind`" + `, a CLI tool that acts as persistent developer memory.
Use it to recall prior context and store new decisions, solutions, and learnings.

### Start of session
` + "```bash" + `
maind recall "<relevant keywords>"
` + "```" + `

### When decisions/bugs/solutions happen
` + "```bash" + `
maind remember "description" --kind decision --tags tag1,tag2 --importance 7
maind remember "bug and fix" --kind solution --tags tag1 --importance 8
maind remember "insight" --kind learning --tags tag1 --importance 6
` + "```" + `

### Commands: remember, recall, list, show, link, tags, forget
### Kinds: note, decision, bug, solution, context, snippet, learning
### Relations: relates_to, caused_by, supersedes, solved_by, depends_on, part_of, derived_from

Always check for prior context before starting work.
Always store meaningful decisions and solutions.
Do not store trivial actions.
`
