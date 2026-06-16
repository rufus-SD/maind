package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rufus-SD/maind/internal/model"
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
	recallJSON    bool
	recallBudget  int
)

func init() {
	recallCmd.Flags().StringVarP(&recallKind, "kind", "k", "", "filter by kind")
	recallCmd.Flags().StringVar(&recallTag, "tag", "", "filter by tag")
	recallCmd.Flags().StringVarP(&recallProject, "project", "p", "", "filter by project")
	recallCmd.Flags().IntVarP(&recallLimit, "limit", "n", 20, "max results")
	recallCmd.Flags().BoolVar(&recallJSON, "json", false, "output full results as JSON (for tools)")
	recallCmd.Flags().IntVar(&recallBudget, "budget", 0, "with --json, cap results to ~N tokens of content")
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

	if recallJSON {
		if recallBudget > 0 {
			entries = trimToTokenBudget(entries, recallBudget)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}

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

// trimToTokenBudget keeps entries (in order) until the estimated token cost of
// their content exceeds budget. At least one entry is always returned so a tiny
// budget never yields an empty result for a matching query.
func trimToTokenBudget(entries []model.Entry, budget int) []model.Entry {
	var out []model.Entry
	used := 0
	for _, e := range entries {
		cost := estimateTokens(e.Body) + estimateTokens(e.Title) + estimateTokens(strings.Join(e.Tags, " "))
		if used+cost > budget && len(out) > 0 {
			break
		}
		out = append(out, e)
		used += cost
	}
	return out
}

// estimateTokens approximates token count at ~4 characters per token.
func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	return (len(s) + 3) / 4
}
