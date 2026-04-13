package sys

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

	destFile, err := os.OpenFile(filepath.Clean(destination), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer destFile.Close()
	if _, err := destFile.Write(input); err != nil {
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
