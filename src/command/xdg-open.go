package command

import (
	"bytes"
	"os/exec"
	"regexp"
	"runtime"

	"github.com/streambinder/spotitube/src/system"
)

// XDGOpenCommand command wrapper implementation
type XDGOpenCommand struct {
	Command
}

// XDGOpen returns a new FFmpegCommand instance
func XDGOpen() XDGOpenCommand {
	return XDGOpenCommand{}
}

// Name returns the effective name of the command
func (c XDGOpenCommand) Name() string {
	if runtime.GOOS == "windows" {
		return "start"
	}

	return "xdg-open"
}

// Exists returns true if the command is installed, false otherwise
func (c XDGOpenCommand) Exists() bool {
	return system.Which(c.Name())
}

// Version returns the command installed version
func (c XDGOpenCommand) Version() string {
	var (
		cmdOut    bytes.Buffer
		cmdErr    error
		cmdReg    *regexp.Regexp
		cmdRegStr = "\\d+\\.\\d+\\.\\d+"
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

// Open triggers a variable input string opening
func (c XDGOpenCommand) Open(input string) error {
	cmd := exec.Command(c.Name(), input)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
