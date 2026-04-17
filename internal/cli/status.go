package cli

import (
	"fmt"
	"os"

	"github.com/rufus-SD/maind/internal/config"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if Maind is ready (session unlocked, config valid)",
	Long: `Check whether Maind is initialized and the encryption session is active.
Returns exit code 0 if ready, 1 otherwise. Useful for AI tools to verify
before running commands.

Examples:
  maind status
  maind status && maind remember "something"`,
	SilenceErrors: true,
	RunE:          runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	dir := resolveDataDir()
	cfg, err := config.Load(dir)
	if err != nil {
		fmt.Println("not_initialized")
		return fmt.Errorf("maind not initialized — run 'maind init'")
	}

	if !cfg.EncryptionEnabled {
		fmt.Println("ready")
		return nil
	}

	if _, err := readSessionKey(); err != nil {
		fmt.Println("locked")
		fmt.Fprintf(os.Stderr, "Session locked — start the dashboard with 'maind' or run 'maind unlock'\n")
		return fmt.Errorf("session locked")
	}

	fmt.Println("ready")
	return nil
}
