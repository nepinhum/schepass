package config

import (
	"errors"
	"os"
	"path/filepath"
)

var errHomeMissing = errors.New("home directory not found")

// DefaultVaultPath returns the default vault location under the user's home.
func DefaultVaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errHomeMissing
	}
	return filepath.Join(home, ".schepass", "vault.bin"), nil
}
