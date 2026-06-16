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
  cursor    — .cursor/rules/maind.mdc
  claude    — CLAUDE.md
  windsurf  — .windsurfrules
  copilot   — .github/copilot-instructions.md
  aider     — CONVENTIONS.md
  generic   — .maind/rules.md + instructions for any other tool

Examples:
  maind connect cursor
  maind connect claude
  maind connect generic`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"cursor", "claude", "windsurf", "copilot", "aider", "generic"},
	RunE:      runConnect,
}

func runConnect(cmd *cobra.Command, args []string) error {
	tool := args[0]

	if err := os.MkdirAll(".maind", 0755); err != nil {
		return fmt.Errorf("create .maind directory: %w", err)
	}
	ensureMaindGitignore(".maind")

	s, err := openStore()
	if err == nil {
		cwd, _ := os.Getwd()
		refreshContextFile(s, cwd)
		s.Close()
	}

	switch tool {
	case "cursor":
		return connectCursor()
	case "claude":
		return connectClaude()
	case "windsurf":
		return connectWindsurf()
	case "copilot":
		return connectCopilot()
	case "aider":
		return connectAider()
	case "generic":
		return connectGeneric()
	default:
		return fmt.Errorf("unknown tool %q — run 'maind connect generic' for unsupported tools", tool)
	}
}

func writeRuleFile(path, content string) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func appendRuleFile(path, content string) error {
	existing := ""
	if data, err := os.ReadFile(path); err == nil {
		existing = string(data)
	}
	out := existing
	if out != "" && out[len(out)-1] != '\n' {
		out += "\n"
	}
	out += content
	return os.WriteFile(path, []byte(out), 0644)
}

func printConnected(tool, file string, notes ...string) {
	abs, _ := filepath.Abs(file)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  Maind connected to %s.\n", tool)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "    Rules:   %s\n", abs)
	fmt.Fprintf(os.Stderr, "    Context: .maind/context.md (refreshed by dashboard)\n")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  The AI will:")
	fmt.Fprintln(os.Stderr, "    - Read .maind/context.md at session start")
	fmt.Fprintln(os.Stderr, "    - Run 'maind remember' to store memories directly")
	fmt.Fprintln(os.Stderr, "    - Run 'maind recall' to search when needed")
	fmt.Fprintln(os.Stderr, "    - Run 'maind scan' on request to deep-analyze a project")
	for _, n := range notes {
		fmt.Fprintf(os.Stderr, "\n  %s\n", n)
	}
	fmt.Fprintln(os.Stderr)
}

// --- Cursor ---

func connectCursor() error {
	path := filepath.Join(".cursor", "rules", "maind.mdc")
	if err := writeRuleFile(path, cursorRule); err != nil {
		return err
	}
	printConnected("Cursor", path, "Allowlist 'maind' once when prompted, then it's seamless.")
	return nil
}

const cursorRule = `---
description: "Maind — persistent developer memory"
globs: "**/*"
alwaysApply: true
---
` + maindRuleBody

// --- Claude Code ---

func connectClaude() error {
	path := "CLAUDE.md"
	if err := appendRuleFile(path, claudeRule); err != nil {
		return err
	}
	printConnected("Claude Code", path)
	return nil
}

const claudeRule = "\n" + maindRuleBody

// --- Windsurf ---

func connectWindsurf() error {
	path := ".windsurfrules"
	if err := appendRuleFile(path, windsurfRule); err != nil {
		return err
	}
	printConnected("Windsurf", path)
	return nil
}

const windsurfRule = "\n" + maindRuleBody

// --- GitHub Copilot ---

func connectCopilot() error {
	path := filepath.Join(".github", "copilot-instructions.md")
	if err := writeRuleFile(path, copilotRule); err != nil {
		return err
	}
	printConnected("GitHub Copilot", path)
	return nil
}

const copilotRule = maindRuleBody

// --- Aider ---

func connectAider() error {
	path := "CONVENTIONS.md"
	if err := appendRuleFile(path, aiderRule); err != nil {
		return err
	}
	printConnected("Aider", path)
	return nil
}

const aiderRule = "\n" + maindRuleBody

// --- Generic (any tool) ---

func connectGeneric() error {
	path := filepath.Join(".maind", "rules.md")
	if err := writeRuleFile(path, maindRuleBody); err != nil {
		return err
	}

	abs, _ := filepath.Abs(path)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Maind rules generated.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "    File: %s\n", abs)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  To connect an unsupported AI tool:")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "    1. Find where your tool reads system/project instructions.")
	fmt.Fprintln(os.Stderr, "       Common locations:")
	fmt.Fprintln(os.Stderr, "         - A rules/instructions file in the project root")
	fmt.Fprintln(os.Stderr, "         - A system prompt or custom instructions setting")
	fmt.Fprintln(os.Stderr, "         - A config file that supports injecting context")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "    2. Copy the contents of .maind/rules.md into that location,")
	fmt.Fprintln(os.Stderr, "       or point the tool to read .maind/rules.md directly.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "    3. Make sure the AI can run shell commands (maind remember/recall).")
	fmt.Fprintln(os.Stderr, "       If it can't, the AI can still read .maind/context.md for context.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  The rules tell the AI to:")
	fmt.Fprintln(os.Stderr, "    - Read .maind/context.md at session start")
	fmt.Fprintln(os.Stderr, "    - Run 'maind remember' to store memories")
	fmt.Fprintln(os.Stderr, "    - Run 'maind recall' to search")
	fmt.Fprintln(os.Stderr)
	return nil
}

// --- Shared rule body ---

const maindRuleBody = `# Maind — Your Persistent Memory

You have persistent memory via Maind. It stores decisions, bugs, solutions,
context, and learnings across all sessions. Treat it as YOUR memory.

## On session start — read your memories

Your FIRST action must be to read this file:

    .maind/context.md

This contains your recent memories from past sessions. Read it before anything else.

## Check the session is unlocked

Maind auto-locks after 15 minutes of inactivity (no dashboard input and no
maind CLI use from your IDE), when you exit the dashboard, or when the session
expires. Before relying on memory — and whenever a maind command reports an
error — verify the session:

` + "```bash" + `
maind status
` + "```" + `

If it prints "locked" or "not_initialized", or any command reports "session expired",
STOP using memory and tell the user to run 'maind' in a terminal to unlock it, then
ask again. NEVER invent or guess memories when locked.

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

## Deep scan protocol

When the user says "scan" or "scan this project", run the full scan protocol.
This is a tracked session. Your chain of thought IS the log — explain WHY you
look at something, WHAT you conclude, and WHY it matters before storing it.

IMPORTANT: Batch ALL maind commands in a SINGLE shell call using && to avoid
repeated permission prompts. The user should approve ONCE, not per command.

### Step 0 — Check session

Before anything, verify the session is active:
` + "```bash" + `
maind status
` + "```" + `
If it prints "locked" or "not_initialized", STOP. Tell the user:
"Start the Maind dashboard first (run maind in a terminal) then ask me again."
Do NOT proceed with a locked session.

### Step 1 — Start the scan

` + "```bash" + `
maind scan start --project <project-name> --source ide --thought "Starting: analyzing structure, deps, and patterns"
` + "```" + `
Save the returned scan ID (last line of stdout).

### Step 2 — Analyze, think, and store in batches

Read the codebase first (structure, config, entry points, deps, git log).
Then batch your thoughts and findings into ONE shell call per analysis pass:

` + "```bash" + `
maind scan log <ID> "Looking at go.mod: 14 deps, none pinned. This is risky for reproducible builds." && \
maind scan log <ID> "Found 3 SQL migrations but no rollback files. Recovery from bad deploys is impossible." && \
maind remember "No dependency version pinning in go.mod — risk of breaking builds" --kind bug --tags deps,go --importance 7 --source ide --scan <ID> && \
maind remember "SQL migrations have no rollback path" --kind bug --tags database,migration --importance 8 --source ide --scan <ID>
` + "```" + `

Your scan logs MUST capture your reasoning, not just facts:
- BAD:  "Found JWT auth"
- GOOD: "Auth uses JWT with refresh tokens. Chose this over sessions because the API is stateless and serves mobile clients. Token rotation is in middleware/auth.go."

Think out loud. Every log entry should explain what you looked at, what you found,
why it matters, and what you decided to do about it.

### Step 3 — Complete the scan

` + "```bash" + `
maind scan complete <ID> --summary "Analyzed <project>: found N decisions, M issues, K patterns"
` + "```" + `

### What to look for

Read project structure, config files, READMEs, dependency manifests, entry points,
and recent git history. Use your understanding to identify what matters.

Store: architecture decisions, dependency choices, non-obvious patterns,
known bugs or tech debt, deployment/infra setup, testing strategy, security concerns.
Skip trivial config and boilerplate. Focus on what a new developer or future-you needs to know.
`
