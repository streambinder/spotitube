package util

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func FileMoveOrCopy(source, destination string, overwrite ...bool) error {
	if _, err := os.Stat(destination); err == nil && !First(overwrite, false) {
		return errors.New("destination already exists")
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
