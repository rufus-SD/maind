package cli

import (
	"fmt"
	"strings"

	"github.com/rufus-SD/maind/internal/store"

	"github.com/spf13/cobra"
)

var recallCmd = &cobra.Command{
	Use:   "recall [query]",
	Short: "Search your memories",
	Long: `Search through stored memories using full-text search.

Examples:
  maind recall "database timeout"
  maind recall "JWT" --kind decision
  maind recall "OOM" --tag go --limit 5`,
	Args: cobra.ExactArgs(1),
	RunE: runRecall,
}

var (
	recallKind    string
	recallTag     string
	recallProject string
	recallLimit   int
)

func init() {
	recallCmd.Flags().StringVarP(&recallKind, "kind", "k", "", "filter by kind")
	recallCmd.Flags().StringVar(&recallTag, "tag", "", "filter by tag")
	recallCmd.Flags().StringVarP(&recallProject, "project", "p", "", "filter by project")
	recallCmd.Flags().IntVarP(&recallLimit, "limit", "n", 20, "max results")
}

func runRecall(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	entries, err := s.SearchEntries(args[0], store.SearchOptions{
		Kind:    recallKind,
		Tag:     recallTag,
		Project: recallProject,
		Limit:   recallLimit,
	})
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	s.LogActivity("RECALL", fmt.Sprintf("%q → %d found", args[0], len(entries)), "")

	if len(entries) == 0 {
		fmt.Println("No memories found.")
		return nil
	}

	for _, e := range entries {
		body := e.Body
		if len(body) > 120 {
			body = body[:120] + "..."
		}
		body = strings.ReplaceAll(body, "\n", " ")

		fmt.Printf("[%s] %s (%d) — %s\n", shortID(e.ID), e.Kind, e.Importance, e.CreatedAt.Format("2006-01-02"))
		if e.Title != "" {
			fmt.Printf("  %s\n", e.Title)
		}
		fmt.Printf("  %s\n", body)
		if len(e.Tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(e.Tags, ", "))
		}
		fmt.Println()
	}

	return nil
}
