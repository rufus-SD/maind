package cli

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/rufus-SD/maind/internal/config"
	"github.com/rufus-SD/maind/internal/crypto"
	storemod "github.com/rufus-SD/maind/internal/store"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Maind — guided setup for your developer memory",
	Long: `Interactive wizard that walks you through setting up Maind:

  1. Name your brain
  2. Choose where to store data
  3. Enable encryption (AES-256-GCM + Argon2id)
  4. Add maind to your PATH

If Maind is already initialized, shows the current configuration and stats.

Examples:
  maind init
  maind init --data-dir ~/my-brain`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	dir := resolveDataDir()
	if config.Exists(dir) {
		return showExistingStatus(dir)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  ┌──────────────────────────────────────┐")
	fmt.Fprintln(os.Stderr, "  │          Welcome to Maind            │")
	fmt.Fprintln(os.Stderr, "  │   Your persistent developer memory   │")
	fmt.Fprintln(os.Stderr, "  └──────────────────────────────────────┘")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Let's set up your second brain.")
	fmt.Fprintln(os.Stderr)

	// --- Step 1: Identity ---
	fmt.Fprintln(os.Stderr, "  ── Step 1/4: Identity ──")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  What should your brain be called?")
	fmt.Fprintln(os.Stderr)
	name := promptLine("  Name", "maind")
	fmt.Fprintln(os.Stderr)

	// --- Step 2: Location ---
	fmt.Fprintln(os.Stderr, "  ── Step 2/4: Storage ──")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Where should Maind store your data?")
	fmt.Fprintln(os.Stderr)

	dir = chooseDataDir()
	fmt.Fprintln(os.Stderr)

	if config.Exists(dir) {
		return showExistingStatus(dir)
	}

	// --- Step 3: Encryption ---
	fmt.Fprintln(os.Stderr, "  ── Step 3/4: Security ──")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Maind can encrypt the content of your memories at rest.")
	fmt.Fprintln(os.Stderr, "  Uses AES-256-GCM with Argon2id key derivation.")
	fmt.Fprintln(os.Stderr, "  Even if someone copies your database, they can't read it.")
	fmt.Fprintln(os.Stderr)

	enableEncrypt := promptYesNo("  Enable encryption?", false)
	fmt.Fprintln(os.Stderr)

	var key []byte
	var salt []byte

	if enableEncrypt {
		passphrase := os.Getenv("MAIND_PASSPHRASE")
		if passphrase == "" {
			fmt.Fprintln(os.Stderr, "  Choose a passphrase. You'll need it every time you use Maind.")
			fmt.Fprintln(os.Stderr, "  Tip: set MAIND_PASSPHRASE env var to skip the prompt.")
			fmt.Fprintln(os.Stderr)

			var err error
			passphrase, err = promptSecret("  Enter passphrase: ")
			if err != nil {
				return fmt.Errorf("read passphrase: %w", err)
			}
			if passphrase == "" {
				return fmt.Errorf("passphrase cannot be empty when encryption is enabled")
			}
			confirm, err := promptSecret("  Confirm passphrase: ")
			if err != nil {
				return fmt.Errorf("read confirmation: %w", err)
			}
			if passphrase != confirm {
				return fmt.Errorf("passphrases do not match")
			}
			fmt.Fprintln(os.Stderr)
		}

		var err error
		salt, err = crypto.GenerateSalt()
		if err != nil {
			return fmt.Errorf("generate salt: %w", err)
		}
		key = crypto.DeriveKey(passphrase, salt)
	}

	// --- Create everything ---
	if err := os.MkdirAll(dir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "\n  Could not create %s: %v\n", dir, err)
		fmt.Fprintln(os.Stderr, "  Please choose a path inside your home directory.")
		fmt.Fprintln(os.Stderr)
		dir = chooseDataDir()
		fmt.Fprintln(os.Stderr)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create data directory: %w", err)
		}
	}

	cfg := &config.Config{
		Version:           1,
		Name:              name,
		EncryptionEnabled: enableEncrypt,
		DBPath:            config.DBFileName,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
	}
	if enableEncrypt {
		cfg.EncryptionSalt = base64.StdEncoding.EncodeToString(salt)
		verifier, err := crypto.CreateVerifier(key)
		if err != nil {
			return fmt.Errorf("create verifier: %w", err)
		}
		cfg.EncryptionVerifier = verifier
	}

	if err := cfg.Save(dir); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	s, err := storemod.New(cfg, dir, key)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer s.Close()

	if err := s.Migrate(); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// --- Step 4: PATH ---
	fmt.Fprintln(os.Stderr, "  ── Step 4/4: PATH ──")
	fmt.Fprintln(os.Stderr)
	offerPathSetup()

	// --- Summary ---
	fmt.Fprintln(os.Stderr, "  ──────────────────────────────────────")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Maind is ready.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "    Name:       %s\n", name)
	fmt.Fprintf(os.Stderr, "    Location:   %s\n", dir)
	fmt.Fprintf(os.Stderr, "    Database:   %s\n", cfg.FullDBPath(dir))
	if enableEncrypt {
		fmt.Fprintln(os.Stderr, "    Encryption: on (AES-256-GCM + Argon2id)")
	} else {
		fmt.Fprintln(os.Stderr, "    Encryption: off")
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Get started:")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "    maind remember \"your first thought\"")
	fmt.Fprintln(os.Stderr, "    maind recall  \"search anything\"")
	fmt.Fprintln(os.Stderr, "    maind list")
	fmt.Fprintln(os.Stderr, "    maind --help")
	fmt.Fprintln(os.Stderr)

	return nil
}

// --- Directory browser ---

func chooseDataDir() string {
	suggestions := config.SuggestedDirs()

	fmt.Fprintln(os.Stderr, "  Suggested locations:")
	for i, s := range suggestions {
		fmt.Fprintf(os.Stderr, "    %d. %s  (%s)\n", i+1, compactPath(s.Path), s.Label)
	}
	fmt.Fprintf(os.Stderr, "    %d. Browse / type a custom path\n", len(suggestions)+1)
	fmt.Fprintln(os.Stderr)

	choice := promptLine("  Choice", "1")

	idx := 0
	if _, err := fmt.Sscanf(choice, "%d", &idx); err == nil && idx >= 1 && idx <= len(suggestions) {
		return suggestions[idx-1].Path
	}

	if idx == len(suggestions)+1 || choice == "browse" || choice == "b" {
		return browseForDir()
	}

	return expandHome(choice)
}

func browseForDir() string {
	home, _ := os.UserHomeDir()
	current := home

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Navigate to your target directory.")
	fmt.Fprintln(os.Stderr, "  Type a folder name to enter it, '..' to go up,")
	fmt.Fprintln(os.Stderr, "  or press Enter to install here.")
	fmt.Fprintln(os.Stderr)

	for {
		fmt.Fprintf(os.Stderr, "  %s/\n", compactPath(current))
		fmt.Fprintln(os.Stderr, "  ──────────────────")

		dirs := listSubDirs(current)
		if len(dirs) == 0 {
			fmt.Fprintln(os.Stderr, "    (empty)")
		} else {
			for _, d := range dirs {
				fmt.Fprintf(os.Stderr, "    %s/\n", d)
			}
		}
		fmt.Fprintln(os.Stderr)

		input := promptLine("  >", "")

		if input == "" {
			final := filepath.Join(current, "maind")
			fmt.Fprintf(os.Stderr, "\n  Data will be stored in: %s\n", compactPath(final))
			return final
		}

		if input == ".." {
			current = filepath.Dir(current)
			fmt.Fprintln(os.Stderr)
			continue
		}

		if strings.HasPrefix(input, "/") || strings.HasPrefix(input, "~") {
			input = expandHome(input)
			if info, err := os.Stat(input); err == nil && info.IsDir() {
				current = input
				fmt.Fprintln(os.Stderr)
				continue
			}
			final := input
			if !strings.HasSuffix(strings.ToLower(filepath.Base(final)), "maind") {
				final = filepath.Join(final, "maind")
			}
			fmt.Fprintf(os.Stderr, "\n  Data will be stored in: %s\n", compactPath(final))
			return final
		}

		target := filepath.Join(current, input)
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			current = target
			fmt.Fprintln(os.Stderr)
		} else {
			fmt.Fprintf(os.Stderr, "  '%s' is not a directory.\n\n", input)
		}
	}
}

func listSubDirs(path string) []string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		dirs = append(dirs, name)
	}
	sort.Strings(dirs)
	if len(dirs) > 20 {
		dirs = dirs[:20]
		dirs = append(dirs, "...")
	}
	return dirs
}

// --- PATH setup ---

func offerPathSetup() {
	binDir := goBinDir()
	if binDir == "" {
		return
	}

	if pathContains(binDir) {
		fmt.Fprintln(os.Stderr, "  maind is already in your PATH. Nothing to do.")
		fmt.Fprintln(os.Stderr)
		return
	}

	fmt.Fprintf(os.Stderr, "  maind binary is at: %s\n", binDir)
	fmt.Fprintln(os.Stderr, "  It's not in your PATH yet.")
	fmt.Fprintln(os.Stderr)

	rcFiles := shellRCFiles()
	if len(rcFiles) == 0 {
		fmt.Fprintf(os.Stderr, "  Add this to your shell profile:\n")
		fmt.Fprintf(os.Stderr, "    export PATH=\"$PATH:%s\"\n\n", binDir)
		return
	}

	fmt.Fprintln(os.Stderr, "  Add to PATH automatically?")
	for i, rc := range rcFiles {
		fmt.Fprintf(os.Stderr, "    %d. %s\n", i+1, compactPath(rc))
	}
	fmt.Fprintf(os.Stderr, "    %d. Skip — I'll do it myself\n", len(rcFiles)+1)
	fmt.Fprintln(os.Stderr)

	choice := promptLine("  Choice", "1")

	idx := 0
	if _, err := fmt.Sscanf(choice, "%d", &idx); err == nil && idx >= 1 && idx <= len(rcFiles) {
		rcFile := rcFiles[idx-1]
		line := fmt.Sprintf("\n# Maind\nexport PATH=\"$PATH:%s\"\n", binDir)

		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Could not write to %s: %v\n\n", rcFile, err)
			return
		}
		f.WriteString(line)
		f.Close()

		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "  Added to %s\n", compactPath(rcFile))
		fmt.Fprintf(os.Stderr, "  Run 'source %s' or restart your terminal.\n", compactPath(rcFile))
	} else {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "  Add this to your shell profile when ready:\n")
		fmt.Fprintf(os.Stderr, "    export PATH=\"$PATH:%s\"\n", binDir)
	}
	fmt.Fprintln(os.Stderr)
}

func goBinDir() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		gopath = filepath.Join(home, "go")
	}
	binDir := filepath.Join(gopath, "bin")
	if _, err := os.Stat(filepath.Join(binDir, "maind")); err != nil {
		exe, err := os.Executable()
		if err != nil {
			return ""
		}
		return filepath.Dir(exe)
	}
	return binDir
}

func pathContains(dir string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == dir {
			return true
		}
	}
	return false
}

func shellRCFiles() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			filepath.Join(home, ".zshrc"),
			filepath.Join(home, ".bash_profile"),
			filepath.Join(home, ".profile"),
		}
	default:
		candidates = []string{
			filepath.Join(home, ".bashrc"),
			filepath.Join(home, ".zshrc"),
			filepath.Join(home, ".profile"),
		}
	}

	var existing []string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			existing = append(existing, c)
		}
	}
	return existing
}

// --- helpers ---

func compactPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func showExistingStatus(dir string) error {
	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Maind is already set up.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "    Name:       %s\n", cfg.Name)
	fmt.Fprintf(os.Stderr, "    Location:   %s\n", compactPath(dir))
	fmt.Fprintf(os.Stderr, "    Database:   %s\n", compactPath(cfg.FullDBPath(dir)))
	if cfg.EncryptionEnabled {
		fmt.Fprintln(os.Stderr, "    Encryption: on")
	} else {
		fmt.Fprintln(os.Stderr, "    Encryption: off")
	}
	fmt.Fprintf(os.Stderr, "    Created:    %s\n", cfg.CreatedAt)

	var key []byte
	if cfg.EncryptionEnabled {
		key, err = resolveEncryptionKey(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "    (could not open database — wrong passphrase?)")
			fmt.Fprintln(os.Stderr)
			return nil
		}
	}

	s, err := storemod.New(cfg, dir, key)
	if err != nil {
		return nil
	}
	defer s.Close()
	if err := s.Migrate(); err != nil {
		return nil
	}

	var entries, archived, tags, links int
	s.DB().QueryRow("SELECT COUNT(*) FROM entries WHERE deleted_at IS NULL").Scan(&entries)
	s.DB().QueryRow("SELECT COUNT(*) FROM entries WHERE deleted_at IS NOT NULL").Scan(&archived)
	s.DB().QueryRow("SELECT COUNT(*) FROM tags").Scan(&tags)
	s.DB().QueryRow("SELECT COUNT(*) FROM links").Scan(&links)

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "    Memories:   %d active, %d archived\n", entries, archived)
	fmt.Fprintf(os.Stderr, "    Tags:       %d\n", tags)
	fmt.Fprintf(os.Stderr, "    Links:      %d\n", links)
	fmt.Fprintln(os.Stderr)

	return nil
}
