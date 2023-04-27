package util

import (
	"os"
)

func FileMoveOrCopy(source, destination string) error {
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
