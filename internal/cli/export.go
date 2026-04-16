package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export all memories as JSON",
	Long: `Export every entry, tag, and link as a JSON document to stdout.
Redirect to a file for backup:

  maind export > backup.json`,
	RunE: runExport,
}

func runExport(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	data, err := s.ExportAll()
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
