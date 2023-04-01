package cmd

import (
	"os/exec"
	"path/filepath"
	"strings"
)

func YouTubeDl(url, path string) error {
	pathExtension := filepath.Ext(path)[1:]
	path = strings.TrimSuffix(path, "."+pathExtension)
	return exec.Command("youtube-dl",
		"--format", "bestaudio",
		"--extract-audio",
		"--audio-format", pathExtension,
		"--audio-quality", "0",
		"--output", path+".%(ext)s",
		"--continue",
		"--no-overwrites",
		url,
	).Run()
}
