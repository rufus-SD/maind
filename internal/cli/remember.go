package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/rufus-SD/maind/internal/model"

	"github.com/spf13/cobra"
)

var rememberCmd = &cobra.Command{
	Use:   "remember [text]",
	Short: "Store a new memory",
	Long: `Save a piece of knowledge, decision, bug, or context.

Examples:
  maind remember "Always use context.WithTimeout for DB calls"
  maind remember "Fixed OOM by switching to streaming JSON parser" --kind solution --tags go,performance --importance 8
  maind remember --title "Auth architecture" --kind decision --project myapp "Use JWT with refresh tokens"`,
	Args: cobra.ExactArgs(1),
	RunE: runRemember,
}

var (
	flagKind       string
	flagTitle      string
	flagTags       string
	flagImportance int
	flagProject    string
	flagSource     string
	flagScanID     string
)

func init() {
	rememberCmd.Flags().StringVarP(&flagKind, "kind", "k", "note", "entry kind (note|decision|bug|solution|context|snippet|learning)")
	rememberCmd.Flags().StringVarP(&flagTitle, "title", "t", "", "short title for the memory")
	rememberCmd.Flags().StringVar(&flagTags, "tags", "", "comma-separated tags")
	rememberCmd.Flags().IntVarP(&flagImportance, "importance", "i", 5, "importance 0-10")
	rememberCmd.Flags().StringVarP(&flagProject, "project", "p", "", "project context")
	rememberCmd.Flags().StringVar(&flagSource, "source", "cli", "source (cli|import|git|ide|api)")
	rememberCmd.Flags().StringVar(&flagScanID, "scan", "", "link this memory to a scan session")
}

func runRemember(cmd *cobra.Command, args []string) error {
	kind := model.EntryKind(flagKind)
	if !model.ValidKinds[kind] {
		return fmt.Errorf("invalid kind %q", flagKind)
	}
	if flagImportance < 0 || flagImportance > 10 {
		return fmt.Errorf("importance must be between 0 and 10")
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	var tags []string
	if flagTags != "" {
		for _, t := range strings.Split(flagTags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	entry := &model.Entry{
		Kind:       kind,
		Title:      flagTitle,
		Body:       args[0],
		Importance: flagImportance,
		Source:     flagSource,
		Project:    flagProject,
		Tags:       tags,
	}

	if err := s.CreateEntry(entry); err != nil {
		return fmt.Errorf("save memory: %w", err)
	}

	if flagScanID != "" {
		resolved, err := s.ResolveScanID(flagScanID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not link to scan %s: %v\n", flagScanID, err)
		} else if err := s.LinkEntryToScan(entry.ID, resolved); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not link to scan: %v\n", err)
		}
	}

	s.LogActivity("STORE", fmt.Sprintf("%s: %s", entry.Kind, truncBody(entry.Body, 60)), entry.ID)

	fmt.Fprintf(os.Stderr, "Saved [%s]\n", shortID(entry.ID))
	fmt.Fprintf(os.Stderr, "  Kind: %s", entry.Kind)
	if len(tags) > 0 {
		fmt.Fprintf(os.Stderr, " | Tags: %s", strings.Join(tags, ", "))
	}
	fmt.Fprintf(os.Stderr, " | Importance: %d\n", entry.Importance)

	fmt.Println(entry.ID)
	return nil
}

func truncBody(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
