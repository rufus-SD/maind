package cli

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Maind version",
	Long:  `Print the current Maind version. Release builds stamp it via ldflags; go install builds fall back to VCS build info.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("maind %s\n", resolveVersion())
	},
}

// resolveVersion prefers the ldflags-stamped Version (release builds). For
// `go install` builds — where Version is the default "dev" — it falls back to
// Go's embedded build info so the output still pins a module version/commit.
func resolveVersion() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	var rev string
	dirty := ""
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "+dirty"
			}
		}
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	mod := info.Main.Version // e.g. v0.2.1 or a pseudo-version; "(devel)" for local builds
	switch {
	case rev != "" && (mod == "" || mod == "(devel)"):
		return "dev (" + rev + dirty + ")"
	case rev != "":
		return mod + " (" + rev + dirty + ")"
	case mod != "" && mod != "(devel)":
		return mod
	default:
		return "dev"
	}
}
