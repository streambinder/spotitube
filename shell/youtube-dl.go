package shell

import (
	"bytes"
	"os/exec"
	"regexp"

	"github.com/streambinder/spotitube/system"
)

// YoutubeDLCommand command wrapper implementation
type YoutubeDLCommand struct {
	Command
}

// YoutubeDL returns a new YoutubeDLCommand instance
func YoutubeDL() YoutubeDLCommand {
	return YoutubeDLCommand{}
}

// Name returns the effective name of the command
func (c YoutubeDLCommand) Name() string {
	return "youtube-dl"
}

// Exists returns true if the command is installed, false otherwise
func (c YoutubeDLCommand) Exists() bool {
	return system.Which(c.Name())
}

// Version returns the command installed version
func (c YoutubeDLCommand) Version() (version string) {
	var (
		cmdOut bytes.Buffer
		cmdReg = regexp.MustCompile("\\d+\\.\\d+\\.\\d+")
	)

	cmd := exec.Command(c.Name(), []string{"--version"}...)
	cmd.Stdout = &cmdOut
	if err := cmd.Run(); err != nil {
		return
	}

	return cmdReg.FindString(cmdOut.String())
}

// Download attempts to download asset at given url
// to given filename using given extension
func (c YoutubeDLCommand) Download(url, filename, extension string) error {
	return exec.Command("youtube-dl", []string{
		"--format", "bestaudio", "--extract-audio",
		"--audio-format", extension,
		"--audio-quality", "0",
		"--output", filename + ".%(ext)s", url}...).Run()
}
