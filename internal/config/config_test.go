package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeBrain(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Version: 1, Name: "test", DBPath: DBFileName}
	if err := cfg.Save(dir); err != nil {
		t.Fatal(err)
	}
}

// An existing brain must be discovered wherever it lives, even if it's not at
// the current per-OS default — this is the Windows init/run mismatch regression.
func TestFindDataDirDiscoversClassicBrain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows home
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("XDG_DATA_HOME", "")

	classic := filepath.Join(home, ".maind")
	writeBrain(t, classic)

	// The default location is empty; the classic one holds the brain.
	if Exists(DefaultDataDir()) {
		t.Skip("default data dir unexpectedly populated in this environment")
	}
	if got := FindDataDir(); got != classic {
		t.Fatalf("FindDataDir() = %q, want %q", got, classic)
	}
}

func TestFindDataDirPrefersDefaultWhenPresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("XDG_DATA_HOME", "")

	def := DefaultDataDir()
	writeBrain(t, def)
	writeBrain(t, filepath.Join(home, ".maind")) // also present, must lose

	if got := FindDataDir(); got != def {
		t.Fatalf("FindDataDir() = %q, want default %q", got, def)
	}
}

func TestFindDataDirFallsBackToDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("XDG_DATA_HOME", "")

	if got := FindDataDir(); got != DefaultDataDir() {
		t.Fatalf("FindDataDir() = %q, want default %q", got, DefaultDataDir())
	}
}
