package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show a single memory with full details",
	Long: `Display full details of a memory entry, including tags and links.
You can use a short ID prefix (at least 8 characters).

Examples:
  maind show a1b2c3d4
  maind show a1b2c3d4-5678-...`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

func runShow(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	entry, err := s.GetEntry(args[0])
	if err != nil {
		return fmt.Errorf("get entry: %w", err)
	}

	fmt.Printf("ID:         %s\n", entry.ID)
	fmt.Printf("Kind:       %s\n", entry.Kind)
	if entry.Title != "" {
		fmt.Printf("Title:      %s\n", entry.Title)
	}
	fmt.Printf("Importance: %d/10\n", entry.Importance)
	fmt.Printf("Source:     %s\n", entry.Source)
	if entry.Project != "" {
		fmt.Printf("Project:    %s\n", entry.Project)
	}
	if len(entry.Tags) > 0 {
		fmt.Printf("Tags:       %s\n", strings.Join(entry.Tags, ", "))
	}
	fmt.Printf("Created:    %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
	fmt.Printf("Updated:    %s\n", entry.UpdatedAt.Format("2006-01-02 15:04:05 UTC"))
	if entry.DeletedAt != nil {
		fmt.Printf("Archived:   %s\n", entry.DeletedAt.Format("2006-01-02 15:04:05 UTC"))
	}
	if entry.BodyEncrypted {
		fmt.Printf("Encrypted:  yes\n")
	}
	fmt.Println()
	fmt.Println(entry.Body)

	links, err := s.GetEntryLinks(entry.ID)
	if err != nil {
		return nil
	}
	if len(links) > 0 {
		fmt.Printf("\nLinks (%d):\n", len(links))
		for _, l := range links {
			if l.FromEntryID == entry.ID {
				fmt.Printf("  -> [%s] %s\n", shortID(l.ToEntryID), l.Relation)
			} else {
				fmt.Printf("  <- [%s] %s\n", shortID(l.FromEntryID), l.Relation)
			}
		}
	}

	return nil
}
