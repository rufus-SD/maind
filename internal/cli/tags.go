package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags with entry counts",
	Long: `Show all tags in your memory, sorted by name, with the number of
entries each tag is attached to. Useful for discovering what topics
you've been storing.

Examples:
  maind tags`,
	RunE: runTags,
}

func runTags(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	tags, err := s.ListTags()
	if err != nil {
		return fmt.Errorf("list tags: %w", err)
	}

	if len(tags) == 0 {
		fmt.Println("No tags yet.")
		return nil
	}

	for _, t := range tags {
		fmt.Printf("%-20s (%d entries)\n", t.Name, t.Count)
	}
	return nil
}
