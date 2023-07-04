package cmd

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/streambinder/spotitube/util"
)

func YouTubeDl(url, path string) error {
	pathExtension := filepath.Ext(path)[1:]
	path = strings.TrimSuffix(util.FileBaseStem(path), "."+pathExtension)
	return exec.Command("yt-dlp",
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
