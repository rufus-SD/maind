package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Maind version",
	Long:  `Print the current Maind version. Set at build time via ldflags.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("maind %s\n", Version)
	},
}
