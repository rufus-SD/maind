package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rufus-SD/maind/internal/model"

	"github.com/spf13/cobra"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest memories from stdin (JSON, one per line)",
	Long: `Read JSON entries from stdin and store them. One JSON object per line.

Each object can have: kind, title, body, tags, importance, project, source.
Only "body" is required.

Examples:
  echo '{"body":"Use JWT for auth","kind":"decision","tags":["auth"],"importance":7}' | maind ingest

  cat entries.jsonl | maind ingest`,
	RunE: runIngest,
}

type ingestEntry struct {
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Tags       []string `json:"tags"`
	Importance *int     `json:"importance"`
	Project    string   `json:"project"`
	Source     string   `json:"source"`
}

func runIngest(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	count := 0
	errors := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var ie ingestEntry
		if err := json.Unmarshal([]byte(line), &ie); err != nil {
			fmt.Fprintf(os.Stderr, "  skip (invalid JSON): %s\n", truncate(line, 60))
			errors++
			continue
		}

		if ie.Body == "" {
			fmt.Fprintf(os.Stderr, "  skip (no body): %s\n", truncate(line, 60))
			errors++
			continue
		}

		kind := model.EntryKind(ie.Kind)
		if ie.Kind == "" {
			kind = model.KindNote
		}
		if !model.ValidKinds[kind] {
			kind = model.KindNote
		}

		importance := 5
		if ie.Importance != nil {
			importance = *ie.Importance
			if importance < 0 {
				importance = 0
			}
			if importance > 10 {
				importance = 10
			}
		}

		source := ie.Source
		if source == "" || source == "cursor" {
			source = "ide"
		}

		entry := &model.Entry{
			Kind:       kind,
			Title:      ie.Title,
			Body:       ie.Body,
			Tags:       ie.Tags,
			Importance: importance,
			Project:    ie.Project,
			Source:     source,
		}

		if err := s.CreateEntry(entry); err != nil {
			fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			errors++
			continue
		}
		s.LogActivity("INGEST", fmt.Sprintf("%s: %s", entry.Kind, truncBody(entry.Body, 50)), entry.ID)

		count++
		fmt.Println(entry.ID)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	fmt.Fprintf(os.Stderr, "  Ingested %d memories", count)
	if errors > 0 {
		fmt.Fprintf(os.Stderr, " (%d errors)", errors)
	}
	fmt.Fprintln(os.Stderr)

	return nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
