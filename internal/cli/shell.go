package cli

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rufus-SD/maind/internal/config"
	"github.com/rufus-SD/maind/internal/crypto"
	"github.com/rufus-SD/maind/internal/model"
	storemod "github.com/rufus-SD/maind/internal/store"

	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start an interactive Maind session",
	Long: `Open a persistent Maind shell. Enter your passphrase once,
then work with your memories interactively. Type /bye or Ctrl-C to lock and exit.

Commands inside the shell:
  remember <text>         Store a memory (prompts for kind, tags, importance)
  recall <query>          Search memories
  list [kind]             List memories
  show <id>               Show a memory
  tags                    List tags
  link <from> <to>        Link two memories
  forget <id>             Archive a memory
  help                    Show available commands
  /bye                    Lock and exit`,
	RunE: runShell,
}

func runShell(cmd *cobra.Command, args []string) error {
	dir := resolveDataDir()
	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("load config: %w\nRun 'maind init' first.", err)
	}

	var key []byte
	if cfg.EncryptionEnabled {
		key, err = unlockForShell(cfg)
		if err != nil {
			return err
		}
	}

	s, err := storemod.New(cfg, dir, key)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer s.Close()

	if err := s.Migrate(); err != nil {
		return fmt.Errorf("migration: %w", err)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  Maind shell — %s\n", cfg.Name)
	fmt.Fprintln(os.Stderr, "  Type 'help' for commands, '/bye' to exit.")
	fmt.Fprintln(os.Stderr)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		fmt.Fprintln(os.Stderr, "\n  Locked. Bye.")
		s.Close()
		os.Exit(0)
	}()

	for {
		fmt.Fprint(os.Stderr, "  maind> ")
		line, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "\n  Locked. Bye.")
			return nil
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "/bye" || line == "/exit" || line == "/quit" {
			fmt.Fprintln(os.Stderr, "  Locked. Bye.")
			return nil
		}

		parts := splitCommand(line)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "help", "?":
			shellHelp()
		case "remember", "r":
			shellRemember(s, parts[1:])
		case "recall", "search", "s":
			shellRecall(s, parts[1:])
		case "list", "ls", "l":
			shellList(s, parts[1:])
		case "show":
			shellShow(s, parts[1:])
		case "tags":
			shellTags(s)
		case "link":
			shellLink(s, parts[1:])
		case "forget":
			shellForget(s, parts[1:])
		default:
			fmt.Fprintf(os.Stderr, "  Unknown command: %s (type 'help')\n", parts[0])
		}
		fmt.Fprintln(os.Stderr)
	}
}

func unlockForShell(cfg *config.Config) ([]byte, error) {
	if key, err := readSessionKey(); err == nil {
		return key, nil
	}

	passphrase := os.Getenv("MAIND_PASSPHRASE")
	if passphrase == "" {
		var err error
		passphrase, err = promptSecret("  Enter passphrase: ")
		if err != nil {
			return nil, fmt.Errorf("read passphrase: %w", err)
		}
		fmt.Fprintln(os.Stderr)
	}
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase required")
	}

	salt, err := base64.StdEncoding.DecodeString(cfg.EncryptionSalt)
	if err != nil {
		return nil, fmt.Errorf("invalid salt: %w", err)
	}

	key := crypto.DeriveKey(passphrase, salt)

	writeSessionKey(key)

	return key, nil
}

func shellHelp() {
	fmt.Fprintln(os.Stderr, "  Commands:")
	fmt.Fprintln(os.Stderr, "    remember <text>         Store a memory")
	fmt.Fprintln(os.Stderr, "    recall <query>          Search memories")
	fmt.Fprintln(os.Stderr, "    list [kind]             List memories")
	fmt.Fprintln(os.Stderr, "    show <id>               Show full detail")
	fmt.Fprintln(os.Stderr, "    tags                    List all tags")
	fmt.Fprintln(os.Stderr, "    link <from> <to>        Link two memories")
	fmt.Fprintln(os.Stderr, "    forget <id>             Archive a memory")
	fmt.Fprintln(os.Stderr, "    /bye                    Lock and exit")
}

func shellRemember(s *storemod.Store, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "  Usage: remember <text>")
		return
	}
	body := strings.Join(args, " ")

	fmt.Fprint(os.Stderr, "  Kind (note/decision/bug/solution/context/snippet/learning) [note]: ")
	kindStr, _ := stdinReader.ReadString('\n')
	kindStr = strings.TrimSpace(kindStr)
	if kindStr == "" {
		kindStr = "note"
	}
	kind := model.EntryKind(kindStr)
	if !model.ValidKinds[kind] {
		kind = model.KindNote
	}

	fmt.Fprint(os.Stderr, "  Tags (comma-separated) []: ")
	tagsStr, _ := stdinReader.ReadString('\n')
	tagsStr = strings.TrimSpace(tagsStr)
	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	fmt.Fprint(os.Stderr, "  Importance 0-10 [5]: ")
	impStr, _ := stdinReader.ReadString('\n')
	impStr = strings.TrimSpace(impStr)
	importance := 5
	if impStr != "" {
		fmt.Sscanf(impStr, "%d", &importance)
	}

	entry := &model.Entry{
		Kind:       kind,
		Body:       body,
		Tags:       tags,
		Importance: importance,
		Source:     "cli",
	}

	if err := s.CreateEntry(entry); err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "  Saved [%s]\n", shortID(entry.ID))
}

func shellRecall(s *storemod.Store, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "  Usage: recall <query>")
		return
	}
	query := strings.Join(args, " ")
	entries, err := s.SearchEntries(query, storemod.SearchOptions{Limit: 10})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		return
	}
	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "  No memories found.")
		return
	}
	for _, e := range entries {
		body := strings.ReplaceAll(e.Body, "\n", " ")
		if len(body) > 100 {
			body = body[:100] + "..."
		}
		fmt.Fprintf(os.Stderr, "  [%s] %s (%d) — %s\n", shortID(e.ID), e.Kind, e.Importance, body)
	}
}

func shellList(s *storemod.Store, args []string) {
	opts := storemod.ListOptions{Limit: 20, SortBy: "created_at", SortOrder: "DESC"}
	if len(args) > 0 {
		opts.Kind = args[0]
	}
	entries, err := s.ListEntries(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		return
	}
	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "  No memories yet.")
		return
	}
	for _, e := range entries {
		title := e.Title
		if title == "" {
			title = e.Body
			if len(title) > 70 {
				title = title[:70] + "..."
			}
			title = strings.ReplaceAll(title, "\n", " ")
		}
		fmt.Fprintf(os.Stderr, "  [%s] %s (%d) — %s\n", shortID(e.ID), e.Kind, e.Importance, title)
	}
}

func shellShow(s *storemod.Store, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "  Usage: show <id>")
		return
	}
	entry, err := s.GetEntry(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "  ID:         %s\n", entry.ID)
	fmt.Fprintf(os.Stderr, "  Kind:       %s\n", entry.Kind)
	if entry.Title != "" {
		fmt.Fprintf(os.Stderr, "  Title:      %s\n", entry.Title)
	}
	fmt.Fprintf(os.Stderr, "  Importance: %d/10\n", entry.Importance)
	if len(entry.Tags) > 0 {
		fmt.Fprintf(os.Stderr, "  Tags:       %s\n", strings.Join(entry.Tags, ", "))
	}
	fmt.Fprintf(os.Stderr, "  Created:    %s\n", entry.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  %s\n", entry.Body)
}

func shellTags(s *storemod.Store) {
	tags, err := s.ListTags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		return
	}
	if len(tags) == 0 {
		fmt.Fprintln(os.Stderr, "  No tags yet.")
		return
	}
	for _, t := range tags {
		fmt.Fprintf(os.Stderr, "  %-20s (%d)\n", t.Name, t.Count)
	}
}

func shellLink(s *storemod.Store, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "  Usage: link <from-id> <to-id>")
		return
	}
	link := &model.Link{
		FromEntryID: args[0],
		ToEntryID:   args[1],
		Relation:    model.RelRelatesTo,
		Weight:      1.0,
	}
	if len(args) > 2 {
		rel := model.LinkRelation(args[2])
		if model.ValidRelations[rel] {
			link.Relation = rel
		}
	}
	if err := s.CreateLink(link); err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "  Linked [%s] --%s--> [%s]\n", shortID(link.FromEntryID), link.Relation, shortID(link.ToEntryID))
}

func shellForget(s *storemod.Store, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "  Usage: forget <id>")
		return
	}
	if err := s.SoftDeleteEntry(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "  Archived [%s]\n", args[0])
}

func splitCommand(line string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(line); i++ {
		c := line[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(c)
			}
		} else if c == '"' || c == '\'' {
			inQuote = true
			quoteChar = c
		} else if c == ' ' || c == '\t' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
