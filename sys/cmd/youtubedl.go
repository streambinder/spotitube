package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/streambinder/spotitube/sys"
)

func YouTubeDl(url, path string) error {
	var (
		output bytes.Buffer
		ext    = filepath.Ext(path)[1:]
		stem   = strings.TrimSuffix(sys.FileBaseStem(path), "."+ext)
		cmd    = exec.Command(
			"yt-dlp",
			"--format", "bestaudio/best",
			"--extract-audio",
			"--audio-format", ext,
			"--audio-quality", "0",
			"--output", stem+".%(ext)s",
			"--continue",
			"--no-overwrites",
			url,
		)
	)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		// a failed run must not leave a partial behind:
		// downloader.Download would pick it up as if complete
		os.Remove(stem + "." + ext)
		return errors.New(output.String())
	}
	return nil
}
