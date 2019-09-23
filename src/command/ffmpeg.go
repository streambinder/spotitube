package command

import (
	"bytes"
	"os/exec"
	"regexp"

	"github.com/streambinder/spotitube/src/system"
)

// FFmpegCommand command wrapper implementation
type FFmpegCommand struct {
	Command
}

// FFmpeg returns a new FFmpegCommand instance
func FFmpeg() FFmpegCommand {
	return FFmpegCommand{}
}

// Name returns the effective name of the command
func (c FFmpegCommand) Name() string {
	return "ffmpeg"
}

// Exists returns true if the command is installed, false otherwise
func (c FFmpegCommand) Exists() bool {
	return system.Which(c.Name())
}

// Version returns the command installed version
func (c FFmpegCommand) Version() string {
	var (
		cmdOut    bytes.Buffer
		cmdErr    error
		cmdReg    *regexp.Regexp
		cmdRegStr = "\\d+\\.\\d+\\.\\d+"
	)

	cmd := exec.Command(c.Name(), []string{"-version"}...)
	cmd.Stdout = &cmdOut
	if cmdErr = cmd.Run(); cmdErr != nil {
		return ""
	}

	if cmdReg, cmdErr = regexp.Compile(cmdRegStr); cmdErr != nil {
		return ""
	}

	return cmdReg.FindString(cmdOut.String())
}
