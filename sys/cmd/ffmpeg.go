package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/streambinder/spotitube/sys"
)

type FFmpegCmd struct{}

func FFmpeg() FFmpegCmd {
	return FFmpegCmd{}
}

// targetIntegratedLoudness is the LUFS target every track is normalized to (Spotify reference)
const targetIntegratedLoudness = -14.0

// maxNegligibleLoudnessDelta is the largest loudness offset skipped as inaudible
const maxNegligibleLoudnessDelta = 0.5 // LU

type Loudness struct {
	Integrated float64 // measured_I, LUFS
	TruePeak   float64 // measured_TP, dBTP
	Range      float64 // measured_LRA, LU
	Threshold  float64 // measured_thresh, LUFS
	Offset     float64 // target offset, dB
}

func (FFmpegCmd) LoudnessDetect(path string) (Loudness, error) {
	var (
		output bytes.Buffer
		cmd    = exec.Command(
			"ffmpeg",
			"-i", path,
			"-af", fmt.Sprintf("loudnorm=I=%.1f:TP=-1.5:LRA=11:print_format=json", targetIntegratedLoudness),
			"-f", "null",
			"-y", "null",
		)
	)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return Loudness{}, errors.New(output.String())
	}
	return parseLoudness(output.String())
}

func parseLoudness(output string) (Loudness, error) {
	// loudnorm prints its JSON summary at the end of the log, with quoted values
	const parseError = "cannot parse loudness measurement for given track"
	start := strings.LastIndex(output, "{")
	if start < 0 {
		return Loudness{}, errors.New(parseError)
	}

	var raw struct {
		Integrated string `json:"measured_I"`
		TruePeak   string `json:"measured_TP"`
		Range      string `json:"measured_LRA"`
		Threshold  string `json:"measured_thresh"`
		Offset     string `json:"offset"`
	}
	if err := json.Unmarshal([]byte(output[start:]), &raw); err != nil {
		return Loudness{}, errors.New(parseError)
	}

	parsed := make([]float64, 0, 5)
	for _, value := range []string{raw.Integrated, raw.TruePeak, raw.Range, raw.Threshold, raw.Offset} {
		volume, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return Loudness{}, errors.New(parseError)
		}
		parsed = append(parsed, volume)
	}
	return Loudness{parsed[0], parsed[1], parsed[2], parsed[3], parsed[4]}, nil
}

func (FFmpegCmd) LoudnessNormalize(path string, loudness Loudness) error {
	if math.Abs(loudness.Integrated-targetIntegratedLoudness) < maxNegligibleLoudnessDelta {
		return nil
	}

	var (
		output bytes.Buffer
		temp   = sys.FileBaseStem(path) + ".norm" + filepath.Ext(path)
		cmd    = exec.Command(
			"ffmpeg", // nolint:gosec
			"-i", path,
			"-af", fmt.Sprintf(
				"loudnorm=I=%.1f:TP=-1.5:LRA=11:measured_I=%.2f:measured_TP=%.2f:measured_LRA=%.2f:measured_thresh=%.2f:offset=%.2f",
				targetIntegratedLoudness,
				loudness.Integrated, loudness.TruePeak, loudness.Range, loudness.Threshold, loudness.Offset,
			),
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
