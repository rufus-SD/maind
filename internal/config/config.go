package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

const (
	DefaultDirName = "maind"
	ConfigFileName = "config.json"
	DBFileName     = "db.sqlite"
)

type Config struct {
	Version              int    `json:"version"`
	Name                 string `json:"name"`
	EncryptionEnabled    bool   `json:"encryption_enabled"`
	EncryptionSalt       string `json:"encryption_salt,omitempty"`
	EncryptionVerifier   string `json:"encryption_verifier,omitempty"`
	DBPath               string `json:"db_path"`
	CreatedAt            string `json:"created_at"`
}

func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".maind")
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", DefaultDirName)
	case "linux":
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, DefaultDirName)
		}
		return filepath.Join(home, ".local", "share", DefaultDirName)
	default:
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, DefaultDirName)
		}
		return filepath.Join(home, ".maind")
	}
}

type SuggestedDir struct {
	Path  string
	Label string
}

func SuggestedDirs() []SuggestedDir {
	home, _ := os.UserHomeDir()
	var dirs []SuggestedDir

	switch runtime.GOOS {
	case "darwin":
		dirs = append(dirs, SuggestedDir{
			filepath.Join(home, "Library", "Application Support", DefaultDirName),
			"macOS standard",
		})
	case "linux":
		xdg := os.Getenv("XDG_DATA_HOME")
		if xdg == "" {
			xdg = filepath.Join(home, ".local", "share")
		}
		dirs = append(dirs, SuggestedDir{
			filepath.Join(xdg, DefaultDirName),
			"XDG standard",
		})
	}

	dirs = append(dirs, SuggestedDir{
		filepath.Join(home, ".maind"),
		"classic",
	})

	return dirs
}

func Load(dataDir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, ConfigFileName))
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Save(dataDir string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, ConfigFileName), data, 0600)
}

func (c *Config) FullDBPath(dataDir string) string {
	if filepath.IsAbs(c.DBPath) {
		return c.DBPath
	}
	return filepath.Join(dataDir, c.DBPath)
}

func Exists(dataDir string) bool {
	_, err := os.Stat(filepath.Join(dataDir, ConfigFileName))
	return err == nil
}
