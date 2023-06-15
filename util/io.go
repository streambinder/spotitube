package util

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
)

func FileMoveOrCopy(source, destination string, overwrite ...bool) error {
	if _, err := os.Stat(destination); err == nil && !First(overwrite, false) {
		return errors.New("destination already exists: " + destination)
	}

	if err := os.Rename(source, destination); err == nil {
		return nil
	}

	input, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	if err := os.WriteFile(destination, input, 0o644); err != nil {
		return err
	}

	return os.Remove(source)
}

func FileBaseStem(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
}

func CacheDirectory() string {
	return ErrWrap(filepath.Join(string(filepath.Separator), "tmp", "spotitube"))(xdg.CacheFile("spotitube"))
}

func CacheFile(filename string) string {
	return filepath.Join(CacheDirectory(), filename)
}
