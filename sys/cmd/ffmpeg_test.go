package cmd

import (
	"os"
	"path/filepath"
	"testing"

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

// fakeFFmpeg installs a fake ffmpeg binary printing the given output and
// exiting with the given code.
func fakeFFmpeg(t *testing.T, output string, code int) {
	t.Helper()
	installFakeBinaries(t, map[string]string{
		ffmpegName: printfScript(output, code),
	})
}

func BenchmarkFFmpeg(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestLoudnessDetect(&testing.T{})
		TestLoudnessNormalize(&testing.T{})
	}
}

func TestLoudnessDetect(t *testing.T) {
	fakeFFmpeg(t, loudnessDetectOutput, 0)

	loudness, err := FFmpeg().LoudnessDetect("/dev/null")
	assert.Nil(t, err)
	assert.Equal(t, Loudness{-23.45, -3.21, 8.12, -34.2, 9.45}, loudness)
}

// newer ffmpeg revisions print input_*/target_offset keys instead of measured_* ones
const loudnessDetectOutputNewFormat = `[Parsed_loudnorm_0 @ 0x741a44001ac0]
{
	"input_i" : "-22.25",
	"input_tp" : "-18.50",
	"input_lra" : "0.00",
	"input_thresh" : "-32.25",
	"output_i" : "-13.97",
	"output_tp" : "-10.26",
	"output_lra" : "0.00",
	"output_thresh" : "-23.97",
	"normalization_type" : "dynamic",
	"target_offset" : "-0.03"
}
[out#0/null @ 0x5eb69227f340] video:0KiB audio:1125KiB subtitle:0KiB other streams:0KiB global headers:0KiB muxing overhead: unknown
size=N/A time=00:00:03.00 bitrate=N/A speed=37.6x elapsed=0:00:00.07`

func TestLoudnessDetectNewFormat(t *testing.T) {
	fakeFFmpeg(t, loudnessDetectOutputNewFormat, 0)

	loudness, err := FFmpeg().LoudnessDetect("/dev/null")
	assert.Nil(t, err)
	assert.Equal(t, Loudness{-22.25, -18.5, 0, -32.25, -0.03}, loudness)
}

func TestLoudnessDetectFFmpegFailure(t *testing.T) {
	fakeFFmpeg(t, "ffmpeg blew up", 1)

	assert.ErrorContains(t, sys.ErrOnly(FFmpeg().LoudnessDetect("/dev/null")), "ffmpeg blew up")
}

func TestLoudnessDetectParseFloatFailure(t *testing.T) {
	// a non-numeric measurement makes the real strconv.ParseFloat fail
	fakeFFmpeg(t, `{
	"measured_I" : "not-a-number",
	"measured_TP" : "-3.21",
	"measured_LRA" : "8.12",
	"measured_thresh" : "-34.20",
	"offset" : "9.45"
}`, 0)

	assert.EqualError(t,
		sys.ErrOnly(FFmpeg().LoudnessDetect("/dev/null")),
		"cannot parse loudness measurement for given track")
}

func TestLoudnessDetectNoJSON(t *testing.T) {
	fakeFFmpeg(t, "no loudness info here", 0)

	assert.EqualError(t,
		sys.ErrOnly(FFmpeg().LoudnessDetect("/dev/null")),
		"cannot parse loudness measurement for given track")
}

func TestLoudnessDetectMalformedJSON(t *testing.T) {
	fakeFFmpeg(t, "some log line { not json", 0)

	assert.EqualError(t,
		sys.ErrOnly(FFmpeg().LoudnessDetect("/dev/null")),
		"cannot parse loudness measurement for given track")
}

func TestLoudnessNormalize(t *testing.T) {
	// the fake re-encode writes the normalized file at the last argument,
	// which is the temp path the production code then renames over the input
	installFakeBinaries(t, map[string]string{
		ffmpegName: "for last; do :; done\nprintf 'normalized' > \"$last\"\n",
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "track.flac")
	assert.Nil(t, os.WriteFile(path, []byte("original"), 0o600))

	assert.Nil(t, FFmpeg().LoudnessNormalize(path, Loudness{Integrated: -23.45}))
	content, err := os.ReadFile(path)
	assert.Nil(t, err)
	assert.Equal(t, "normalized", string(content))
	entries, err := os.ReadDir(dir)
	assert.Nil(t, err)
	assert.Len(t, entries, 1) // the temp file was renamed, not left behind
}

func TestLoudnessNormalizeSkipped(t *testing.T) {
	// no ffmpeg on PATH at all: any attempt to run it would fail, so a nil
	// result proves inaudible offsets skip the re-encode entirely
	installEmptyPath(t)

	assert.Nil(t, FFmpeg().LoudnessNormalize("/dev/null", Loudness{Integrated: -14.2}))
}

func TestLoudnessNormalizeRenameFailure(t *testing.T) {
	// the fake exits successfully but produces no output file,
	// so the real os.Rename of the temp file fails
	fakeFFmpeg(t, "", 0)

	path := filepath.Join(t.TempDir(), "track.flac")
	err := FFmpeg().LoudnessNormalize(path, Loudness{Integrated: -23.45})
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoudnessNormalizeFFmpegFailure(t *testing.T) {
	fakeFFmpeg(t, "ffmpeg blew up", 1)

	assert.ErrorContains(t,
		FFmpeg().LoudnessNormalize("/dev/null", Loudness{Integrated: -23.45}),
		"ffmpeg blew up")
}
