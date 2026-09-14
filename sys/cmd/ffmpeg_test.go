package cmd

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

const loudnessDetectOutput = `ffmpeg version 5.1.2 Copyright (c) 2000-2022 the FFmpeg developers
[Parsed_loudnorm_0 @ 0x6000036482c0]
{
	"input_i" : "-23.45",
	"input_tp" : "-3.21",
	"input_lra" : "8.12",
	"input_thresh" : "-34.20",
	"output_i" : "-14.00",
	"output_tp" : "-1.50",
	"output_lra" : "7.80",
	"output_thresh" : "-24.43",
	"normalization_type" : "linear",
	"target_offset" : "9.45",
	"measured_I" : "-23.45",
	"measured_TP" : "-3.21",
	"measured_LRA" : "8.12",
	"measured_thresh" : "-34.20",
	"offset" : "9.45"
}`

func BenchmarkFFmpeg(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestLoudnessDetect(&testing.T{})
		TestLoudnessNormalize(&testing.T{})
	}
}

func TestLoudnessDetect(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).To(func(cmd *exec.Cmd) error {
		return sys.ErrOnly(cmd.Stdout.Write([]byte(loudnessDetectOutput)))
	}).Build()

	// testing
	loudness, err := FFmpeg().LoudnessDetect("/dev/null")
	assert.Nil(t, err)
	assert.Equal(t, Loudness{-23.45, -3.21, 8.12, -34.2, 9.45}, loudness)
}

func TestLoudnessDetectFFmpegFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).Return(errors.New("ko")).Build()

	// testing
	assert.Error(t, sys.ErrOnly(FFmpeg().LoudnessDetect("/dev/null")))
}

func TestLoudnessDetectParseFloatFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).To(func(cmd *exec.Cmd) error {
		return sys.ErrOnly(cmd.Stdout.Write([]byte(loudnessDetectOutput)))
	}).Build()
	mockey.Mock(strconv.ParseFloat).Return(0.0, errors.New("ko")).Build()

	// testing
	assert.EqualError(t,
		sys.ErrOnly(FFmpeg().LoudnessDetect("/dev/null")),
		"cannot parse loudness measurement for given track")
}

func TestLoudnessDetectNoJSON(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).To(func(cmd *exec.Cmd) error {
		return sys.ErrOnly(cmd.Stdout.Write([]byte("no loudness info here")))
	}).Build()

	// testing
	assert.EqualError(t,
		sys.ErrOnly(FFmpeg().LoudnessDetect("/dev/null")),
		"cannot parse loudness measurement for given track")
}

func TestLoudnessDetectMalformedJSON(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).To(func(cmd *exec.Cmd) error {
		return sys.ErrOnly(cmd.Stdout.Write([]byte("some log line { not json")))
	}).Build()

	// testing
	assert.EqualError(t,
		sys.ErrOnly(FFmpeg().LoudnessDetect("/dev/null")),
		"cannot parse loudness measurement for given track")
}

func TestLoudnessNormalize(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).Return(nil).Build()
	mockey.Mock(os.Rename).Return(nil).Build()

	// testing
	assert.Nil(t, FFmpeg().LoudnessNormalize("/dev/null", Loudness{Integrated: -23.45}))
}

func TestLoudnessNormalizeSkipped(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).Return(errors.New("must not run")).Build()

	// testing: inaudible offsets skip the re-encode entirely
	assert.Nil(t, FFmpeg().LoudnessNormalize("/dev/null", Loudness{Integrated: -14.2}))
}

func TestLoudnessNormalizeRenameFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).Return(nil).Build()
	mockey.Mock(os.Rename).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, FFmpeg().LoudnessNormalize("/dev/null", Loudness{Integrated: -23.45}), "ko")
}

func TestLoudnessNormalizeFFmpegFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).Return(errors.New("ko")).Build()

	// testing
	assert.Error(t, FFmpeg().LoudnessNormalize("/dev/null", Loudness{Integrated: -23.45}))
}
