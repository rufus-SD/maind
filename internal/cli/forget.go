package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var forgetCmd = &cobra.Command{
	Use:   "forget [id]",
	Short: "Archive a memory (soft delete)",
	Long: `Mark a memory as archived. It is not permanently deleted and can still
be seen with 'maind list --all' or 'maind export'.

Examples:
  maind forget a1b2c3d4`,
	Args: cobra.ExactArgs(1),
	RunE: runForget,
}

func runForget(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	if err := s.SoftDeleteEntry(args[0]); err != nil {
		return fmt.Errorf("archive: %w", err)
	}
	s.LogActivity("ARCHIVE", args[0], args[0])

	fmt.Fprintf(os.Stderr, "Archived [%s]\n", args[0])
	return nil
}
