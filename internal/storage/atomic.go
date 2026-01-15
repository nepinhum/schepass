package storage

import (
	"errors"
	"os"
	"path/filepath"
)

var errEmptyPath = errors.New("empty path")

// WriteFileAtomic writes data to a temp file and renames it into place.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if path == "" {
		return errEmptyPath
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".vault.*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	defer func() {
		tmp.Close()
		os.Remove(tmpName)
	}()

	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
