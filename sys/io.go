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

	// stage the copy on a temp file in the destination directory
	// and rename it atomically: a crash must never leave a
	// partial file behind at the final destination
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".spotitube-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// the temp staging must never survive a failed copy
	staged := false
	defer func() {
		if !staged {
			os.Remove(tmpName)
		}
	}()

	_, err = tmp.Write(input)
	if err == nil {
		err = tmp.Close()
	}
	if err != nil {
		ErrSuppress(tmp.Close())
		return err
	}
	if err := os.Rename(tmpName, destination); err != nil {
		return err
	}
	staged = true

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
