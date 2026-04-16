package cli

import (
	"fmt"
	"strings"

	"github.com/rufus-SD/maind/internal/store"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored memories",
	Long: `List memories with optional filters and sorting.

Examples:
  maind list
  maind list --kind decision --limit 10
  maind list --tag go --sort importance`,
	RunE: runList,
}

var (
	listKind    string
	listTag     string
	listProject string
	listLimit   int
	listSort    string
	listOrder   string
	listAll     bool
)

func init() {
	listCmd.Flags().StringVarP(&listKind, "kind", "k", "", "filter by kind")
	listCmd.Flags().StringVar(&listTag, "tag", "", "filter by tag")
	listCmd.Flags().StringVarP(&listProject, "project", "p", "", "filter by project")
	listCmd.Flags().IntVarP(&listLimit, "limit", "n", 30, "max results")
	listCmd.Flags().StringVarP(&listSort, "sort", "s", "created_at", "sort by (created_at|updated_at|importance|kind)")
	listCmd.Flags().StringVar(&listOrder, "order", "DESC", "sort order (ASC|DESC)")
	listCmd.Flags().BoolVar(&listAll, "all", false, "include archived entries")
}

func runList(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	entries, err := s.ListEntries(store.ListOptions{
		Kind:           listKind,
		Tag:            listTag,
		Project:        listProject,
		Limit:          listLimit,
		SortBy:         listSort,
		SortOrder:      strings.ToUpper(listOrder),
		IncludeDeleted: listAll,
	})
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No memories found.")
		return nil
	}

	for _, e := range entries {
		label := string(e.Kind)
		if e.DeletedAt != nil {
			label += " [archived]"
		}

		title := e.Title
		if title == "" {
			title = e.Body
			if len(title) > 80 {
				title = title[:80] + "..."
			}
			title = strings.ReplaceAll(title, "\n", " ")
		}

		fmt.Printf("[%s] %s (%d) — %s — %s\n", shortID(e.ID), label, e.Importance, e.CreatedAt.Format("2006-01-02"), title)
	}

	return nil
}
