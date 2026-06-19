package cli

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/rufus-SD/maind/internal/config"
	"github.com/rufus-SD/maind/internal/crypto"
	storemod "github.com/rufus-SD/maind/internal/store"
	"github.com/rufus-SD/maind/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	flagDataDir string
	Version     = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "maind",
	Short: "Your second brain — persistent developer memory",
	Long: `Maind is a local CLI that acts as your persistent developer memory.
It stores decisions, bugs, solutions, and context across all your projects.
Your data stays local, encrypted, and private.

Run 'maind' with no subcommand to start the live dashboard.`,
	SilenceUsage: true,
	RunE:         runDashboard,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagDataDir, "data-dir", "", "data directory (default ~/.maind)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(demoCmd)
	rootCmd.AddCommand(rememberCmd)
	rootCmd.AddCommand(recallCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(linkCmd)
	rootCmd.AddCommand(forgetCmd)
	rootCmd.AddCommand(tagsCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(ingestCmd)
	rootCmd.AddCommand(unlockCmd)
	rootCmd.AddCommand(lockCmd)
	rootCmd.AddCommand(passwdCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(versionCmd)
}

func resolveDataDir() string {
	if flagDataDir != "" {
		return flagDataDir
	}
	return config.DefaultDataDir()
}

func openStore() (*storemod.Store, error) {
	dir := resolveDataDir()
	cfg, err := config.Load(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot load config from %s: %w\nRun 'maind init' first.", dir, err)
	}
	var key []byte
	if cfg.EncryptionEnabled {
		key, err = resolveEncryptionKey(cfg)
		if err != nil {
			return nil, err
		}
	}
	s, err := storemod.New(cfg, dir, key)
	if err != nil {
		return nil, err
	}
	if err := s.Migrate(); err != nil {
		s.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	if cfg.EncryptionEnabled {
		_ = touchSessionActivity()
	}
	return s, nil
}

func resolveEncryptionKey(cfg *config.Config) ([]byte, error) {
	if key, err := readSessionKey(); err == nil {
		return key, nil
	}

	passphrase := os.Getenv("MAIND_PASSPHRASE")
	if passphrase == "" {
		var err error
		passphrase, err = promptSecret("  Enter passphrase: ")
		if err != nil {
			return nil, fmt.Errorf("read passphrase: %w", err)
		}
	}
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase required — run 'maind unlock' or set MAIND_PASSPHRASE")
	}
	salt, err := base64.StdEncoding.DecodeString(cfg.EncryptionSalt)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption salt in config: %w", err)
	}
	key := crypto.DeriveKey(passphrase, salt)

	if cfg.EncryptionVerifier != "" && !crypto.CheckVerifier(cfg.EncryptionVerifier, key) {
		return nil, fmt.Errorf("wrong passphrase")
	}

	return key, nil
}

// --- prompt helpers ---

var stdinReader = bufio.NewReader(os.Stdin)

func promptLine(prompt string, defaultVal string) string {
	if defaultVal != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", prompt)
	}
	line, _ := stdinReader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func promptYesNo(prompt string, defaultYes bool) bool {
	hint := "y/N"
	if defaultYes {
		hint = "Y/n"
	}
	fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, hint)
	line, _ := stdinReader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes
	}
	return line == "y" || line == "yes"
}

func promptSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

func runDashboard(cmd *cobra.Command, args []string) error {
	dir := resolveDataDir()
	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("cannot load config from %s: %w\nRun 'maind init' first.", dir, err)
	}

	var key []byte
	if cfg.EncryptionEnabled {
		passphrase := os.Getenv("MAIND_PASSPHRASE")
		if passphrase == "" {
			passphrase, err = promptSecret("  Enter passphrase: ")
			if err != nil {
				return fmt.Errorf("read passphrase: %w", err)
			}
			fmt.Fprintln(os.Stderr)
		}
		if passphrase == "" {
			return fmt.Errorf("passphrase required")
		}
		salt, err := base64.StdEncoding.DecodeString(cfg.EncryptionSalt)
		if err != nil {
			return fmt.Errorf("invalid salt: %w", err)
		}
		key = crypto.DeriveKey(passphrase, salt)

		if cfg.EncryptionVerifier != "" && !crypto.CheckVerifier(cfg.EncryptionVerifier, key) {
			return fmt.Errorf("wrong passphrase")
		}

		if cfg.EncryptionVerifier == "" {
			if v, vErr := crypto.CreateVerifier(key); vErr == nil {
				cfg.EncryptionVerifier = v
				cfg.Save(dir)
			}
		}

		if err := writeSessionKey(key); err != nil {
			return fmt.Errorf("write session key: %w", err)
		}
		defer func() {
			os.Remove(sessionKeyPath())
		}()
	}

	s, err := storemod.New(cfg, dir, key)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer s.Close()

	if err := s.Migrate(); err != nil {
		return fmt.Errorf("migration: %w", err)
	}

	s.LogActivity("SESSION", "Dashboard opened", "")

	refresher := newContextRefresher(s)
	for _, proj := range findConnectedProjects() {
		refresher.addProject(proj)
	}
	refresher.start()
	defer func() {
		refresher.close()
		refresher.cleanup()
	}()

	m := tui.NewModel(s, cfg.Name, cfg.EncryptionEnabled)
	if cfg.EncryptionEnabled {
		m = m.WithSessionHooks(
			func() bool { _, err := readSessionKey(); return err == nil },
			func() error { return refreshSessionKey(key) },
			func() error { return os.Remove(sessionKeyPath()) },
			sessionIdleSince,
			touchSessionActivity,
		)
	}
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("dashboard: %w", err)
	}

	s.LogActivity("SESSION", "Dashboard closed", "")
	reason := ""
	if fm, ok := finalModel.(tui.Model); ok {
		reason = fm.LockReason()
	}
	switch reason {
	case "idle":
		fmt.Fprintln(os.Stderr, "  Auto-locked after 2 hours of inactivity. Bye.")
	case "locked":
		fmt.Fprintln(os.Stderr, "  Session locked elsewhere. Bye.")
	default:
		fmt.Fprintln(os.Stderr, "  Locked. Bye.")
	}
	return nil
}

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}
