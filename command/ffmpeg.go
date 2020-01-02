package command

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/streambinder/spotitube/system"
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

// VolumeDetect returns the float64 volume detection value for a given filename
func (c FFmpegCommand) VolumeDetect(filename string) (float64, error) {
	var (
		cmdOut    bytes.Buffer
		cmdErr    error
		cmdReg    *regexp.Regexp
		cmdRegStr = `max_volume:\s(?P<max_volume>[\-\.0-9]+)\sdB`
	)

	cmd := exec.Command(c.Name(), []string{
		"-i", filename,
		"-af", "volumedetect",
		"-f", "null",
		"-y", "null"}...)
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdOut
	if cmdErr = cmd.Run(); cmdErr != nil {
		return 0.0, cmdErr
	}

	if cmdReg, cmdErr = regexp.Compile(cmdRegStr); cmdErr != nil {
		return 0.0, cmdErr
	}

	cmdRegMatch := cmdReg.FindStringSubmatch(cmdOut.String())
	cmdRegMap := system.MapGroups(cmdRegMatch, cmdReg.SubexpNames())
	if val, ok := cmdRegMap["max_volume"]; ok {
		valFloat, _ := strconv.ParseFloat(val, 64)
		return valFloat, nil
	}

	return 0.0, fmt.Errorf("Max volume value not found")
}

// VolumeSet increases max volume value by a given delta for a fiven filename
func (c FFmpegCommand) VolumeSet(delta float64, filename string) error {
	var (
		cmdOut      bytes.Buffer
		cmdErr      error
		tmpFilename = fmt.Sprintf("/tmp/%s", filepath.Base(filename))
	)

	cmd := exec.Command(c.Name(), []string{
		"-i", filename,
		"-af", fmt.Sprintf("volume=+%fdB", delta),
		"-b:a", "320k",
		"-y", tmpFilename}...)
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdOut
	if cmdErr = cmd.Run(); cmdErr != nil {
		return cmdErr
	}

	cmdErr = system.FileMove(tmpFilename, filename)
	if cmdErr != nil {
		return cmdErr
	}

	return nil
}
