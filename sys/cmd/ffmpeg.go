package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/streambinder/spotitube/sys"
)

type FFmpegCmd struct{}

func FFmpeg() FFmpegCmd {
	return FFmpegCmd{}
}

func (FFmpegCmd) VolumeDetect(path string) (float64, error) {
	var (
		output bytes.Buffer
		regex  = regexp.MustCompile(`max_volume:\s[\-\.0-9]+\sdB`)
		cmd    = exec.Command("ffmpeg",
			"-i", path,
			"-af", "volumedetect",
			"-f", "null",
			"-y", "null",
		)
	)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		// errLines := strings.Split(output.String(), "\n")
		return 0, errors.New(output.String())
	}

	match := regex.FindString(output.String())
	match = strings.ReplaceAll(match, "max_volume: ", "")
	match = strings.ReplaceAll(match, " dB", "")
	volume, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0, errors.New("cannot parse max_volume for given track")
	}
	return volume, nil
}

func (FFmpegCmd) VolumeAdd(path string, delta float64) error {
	if delta == 0 {
		return nil
	}

	var (
		output bytes.Buffer
		temp   = sys.FileBaseStem(path) + ".norm" + filepath.Ext(path)
		cmd    = exec.Command("ffmpeg", // nolint:gosec
			"-i", path,
			"-af", fmt.Sprintf("volume=%.1fdB", math.Abs(delta)),
			"-y", temp,
		)
	)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return errors.New(output.String())
	}
	return os.Rename(temp, path)
}
