package cli

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/rufus-SD/maind/internal/config"
	"github.com/rufus-SD/maind/internal/crypto"
	storemod "github.com/rufus-SD/maind/internal/store"

	"github.com/spf13/cobra"
)

var passwdCmd = &cobra.Command{
	Use:   "passwd",
	Short: "Change your passphrase (re-encrypts all memories)",
	Long: `Rotate the encryption passphrase.

Every memory body and scan note is re-encrypted under the new key in a single
transaction. Requires the current session to be unlocked, or the current
passphrase. Set MAIND_NEW_PASSPHRASE to run non-interactively.`,
	RunE: runPasswd,
}

func runPasswd(cmd *cobra.Command, args []string) error {
	dir := resolveDataDir()
	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("load config: %w\nRun 'maind init' first.", err)
	}
	if !cfg.EncryptionEnabled {
		return fmt.Errorf("encryption is not enabled for this brain")
	}

	// Current key: from the active session, or by prompting the current passphrase.
	oldKey, err := resolveEncryptionKey(cfg)
	if err != nil {
		return err
	}

	newPass := os.Getenv("MAIND_NEW_PASSPHRASE")
	if newPass == "" {
		newPass, err = promptSecret("  New passphrase: ")
		if err != nil {
			return fmt.Errorf("read new passphrase: %w", err)
		}
		confirm, err := promptSecret("  Confirm new passphrase: ")
		if err != nil {
			return fmt.Errorf("read confirmation: %w", err)
		}
		if newPass != confirm {
			return fmt.Errorf("passphrases do not match")
		}
	}
	if newPass == "" {
		return fmt.Errorf("new passphrase cannot be empty")
	}

	newSalt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}
	newKey := crypto.DeriveKey(newPass, newSalt)

	s, err := storemod.New(cfg, dir, oldKey)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := s.Migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if err := s.Reencrypt(newKey); err != nil {
		return fmt.Errorf("re-encrypt memories: %w", err)
	}

	// Data is now under newKey; update config to match, then refresh the session.
	verifier, err := crypto.CreateVerifier(newKey)
	if err != nil {
		return fmt.Errorf("create verifier: %w", err)
	}
	cfg.EncryptionSalt = base64.StdEncoding.EncodeToString(newSalt)
	cfg.EncryptionVerifier = verifier
	if err := cfg.Save(dir); err != nil {
		return fmt.Errorf("CRITICAL: memories were re-encrypted under the NEW passphrase, "+
			"but saving the config failed: %w\nManually set encryption_salt/encryption_verifier "+
			"for the new passphrase, or restore from a backup", err)
	}

	if err := writeSessionKey(newKey); err != nil {
		// Non-fatal: the rotation succeeded; the user can just unlock again.
		fmt.Fprintln(os.Stderr, "  (Could not refresh the session key — run 'maind unlock' with the new passphrase.)")
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Passphrase changed. All memories re-encrypted under the new key.")
	fmt.Fprintln(os.Stderr)
	return nil
}
