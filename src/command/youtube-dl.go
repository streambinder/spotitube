package command

import (
	"bytes"
	"os/exec"
	"regexp"

	"github.com/streambinder/spotitube/src/system"
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
func (c YoutubeDLCommand) Version() string {
	var (
		cmdOut    bytes.Buffer
		cmdErr    error
		cmdReg    *regexp.Regexp
		cmdRegStr = "\\d{4}\\.\\d{2}\\.\\d{2}"
	)

	cmd := exec.Command(c.Name(), []string{"--version"}...)
	cmd.Stdout = &cmdOut
	if cmdErr = cmd.Run(); cmdErr != nil {
		return ""
	}

	if cmdReg, cmdErr = regexp.Compile(cmdRegStr); cmdErr != nil {
		return ""
	}

	return cmdReg.FindString(cmdOut.String())
}
