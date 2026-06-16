package cli

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rufus-SD/maind/internal/config"
	"github.com/rufus-SD/maind/internal/crypto"

	"github.com/spf13/cobra"
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock Maind for this session (no passphrase needed until lock)",
	Long: `Derive the encryption key from your passphrase and cache it locally
so every subsequent maind command works without prompting.

The session key is stored in a temp file readable only by your user.
Run 'maind lock' to end the session.`,
	RunE: runUnlock,
}

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock Maind — clear the cached session key",
	Long: `Delete the cached session key so all subsequent commands require
the passphrase again. The dashboard does this automatically on exit.

Examples:
  maind lock`,
	RunE: runLock,
}

func runUnlock(cmd *cobra.Command, args []string) error {
	dir := resolveDataDir()
	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("load config: %w\nRun 'maind init' first.", err)
	}

	if !cfg.EncryptionEnabled {
		fmt.Fprintln(os.Stderr, "  Encryption is not enabled. No unlock needed.")
		return nil
	}

	passphrase := os.Getenv("MAIND_PASSPHRASE")
	if passphrase == "" {
		var err error
		passphrase, err = promptSecret("  Enter passphrase: ")
		if err != nil {
			return fmt.Errorf("read passphrase: %w", err)
		}
	}
	if passphrase == "" {
		return fmt.Errorf("passphrase cannot be empty")
	}

	salt, err := base64.StdEncoding.DecodeString(cfg.EncryptionSalt)
	if err != nil {
		return fmt.Errorf("invalid salt in config: %w", err)
	}

	key := crypto.DeriveKey(passphrase, salt)

	if cfg.EncryptionVerifier != "" && !crypto.CheckVerifier(cfg.EncryptionVerifier, key) {
		return fmt.Errorf("wrong passphrase")
	}

	if cfg.EncryptionVerifier == "" {
		verifier, err := crypto.CreateVerifier(key)
		if err == nil {
			cfg.EncryptionVerifier = verifier
			cfg.Save(dir)
		}
	}

	if err := writeSessionKey(key); err != nil {
		return fmt.Errorf("cache session key: %w", err)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Unlocked.")
	fmt.Fprintln(os.Stderr, "  All maind commands will work without a passphrase.")
	fmt.Fprintln(os.Stderr, "  Run 'maind lock' when you're done.")
	fmt.Fprintln(os.Stderr)
	return nil
}

func runLock(cmd *cobra.Command, args []string) error {
	path := sessionKeyPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove session key: %w", err)
	}
	fmt.Fprintln(os.Stderr, "  Locked. Passphrase required for next use.")
	return nil
}

func sessionKeyPath() string {
	tmpDir := os.TempDir()
	uid := strconv.Itoa(os.Getuid())
	return filepath.Join(tmpDir, "maind-session-"+uid)
}

func writeSessionKey(key []byte) error {
	path := sessionKeyPath()
	now := time.Now()
	content := fmt.Sprintf("%d\n%s\n%d",
		now.Add(12*time.Hour).Unix(),
		base64.StdEncoding.EncodeToString(key),
		now.Unix(),
	)
	return os.WriteFile(path, []byte(content), 0600)
}

// refreshSessionKey extends the 12-hour session expiry without bumping the idle
// timer — only real use (CLI commands, dashboard keys) should reset inactivity.
func refreshSessionKey(key []byte) error {
	path := sessionKeyPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return writeSessionKey(key)
	}
	lines := splitLines(string(data))
	if len(lines) < 2 {
		return writeSessionKey(key)
	}
	lines[0] = fmt.Sprintf("%d", time.Now().Add(12*time.Hour).Unix())
	if len(lines) < 3 {
		lines = append(lines, fmt.Sprintf("%d", time.Now().Unix()))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
}

// touchSessionActivity bumps the idle timer — call on any successful maind use
// (dashboard keys, remember, recall, etc.) so inactivity reflects brain use, not
// just typing in the TUI.
func touchSessionActivity() error {
	path := sessionKeyPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := splitLines(string(data))
	if len(lines) < 2 {
		return fmt.Errorf("invalid session file")
	}
	if len(lines) >= 3 {
		lines[2] = fmt.Sprintf("%d", time.Now().Unix())
	} else {
		lines = append(lines, fmt.Sprintf("%d", time.Now().Unix()))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
}

// sessionIdleSince returns how long since the last recorded activity.
// Activity is updated by the dashboard and by external CLI commands.
func sessionIdleSince() (time.Duration, error) {
	path := sessionKeyPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	lines := splitLines(string(data))
	if len(lines) < 2 {
		return 0, fmt.Errorf("invalid session file")
	}

	var lastActive time.Time
	if len(lines) >= 3 {
		ts, err := strconv.ParseInt(lines[2], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid activity timestamp")
		}
		lastActive = time.Unix(ts, 0)
	} else {
		// Legacy 2-line session files: treat as just refreshed.
		lastActive = time.Now()
	}

	return time.Since(lastActive), nil
}

func readSessionKey() ([]byte, error) {
	path := sessionKeyPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := splitLines(string(data))
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid session file")
	}

	expiry, err := strconv.ParseInt(lines[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid session expiry")
	}
	if time.Now().Unix() > expiry {
		os.Remove(path)
		return nil, fmt.Errorf("session expired — run 'maind unlock' again")
	}

	return base64.StdEncoding.DecodeString(lines[1])
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
