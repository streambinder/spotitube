package cmd

import (
	"fmt"
	"os/exec"
)

func ValidateEnvironment() error {
	for _, cmd := range []string{"ffmpeg", "yt-dlp"} {
		_, err := exec.LookPath(cmd)
		if err != nil {
			return fmt.Errorf("command %q not found in PATH", cmd)
		}
	}
	return nil
}
