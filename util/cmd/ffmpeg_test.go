package cmd

import (
	"errors"
	"os/exec"
	"reflect"
	"strconv"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

const volumeDetectOutput = `ffmpeg version 5.1.2 Copyright (c) 2000-2022 the FFmpeg developers
built with Apple clang version 14.0.0 (clang-1400.0.29.202)
configuration: --prefix=/Users/dpucci/.local/share/homebrew/Cellar/ffmpeg/5.1.2_6 --enable-shared --enable-pthreads --enable-version3 --cc=clang --host-cflags= --host-ldflags= --enable-ffplay --enable-gnutls --enable-gpl --enable-libaom --enable-libaribb24 --enable-libbluray --enable-libdav1d --enable-libmp3lame --enable-libopus --enable-librav1e --enable-librist --enable-librubberband --enable-libsnappy --enable-libsrt --enable-libsvtav1 --enable-libtesseract --enable-libtheora --enable-libvidstab --enable-libvmaf --enable-libvorbis --enable-libvpx --enable-libwebp --enable-libx264 --enable-libx265 --enable-libxml2 --enable-libxvid --enable-lzma --enable-libfontconfig --enable-libfreetype --enable-frei0r --enable-libass --enable-libopencore-amrnb --enable-libopencore-amrwb --enable-libopenjpeg --enable-libspeex --enable-libsoxr --enable-libzmq --enable-libzimg --disable-libjack --disable-indev=jack --enable-videotoolbox --enable-neon
libavutil      57. 28.100 / 57. 28.100
libavcodec     59. 37.100 / 59. 37.100
libavformat    59. 27.100 / 59. 27.100
libavdevice    59.  7.100 / 59.  7.100
libavfilter     8. 44.100 /  8. 44.100
libswscale      6.  7.100 /  6.  7.100
libswresample   4.  7.100 /  4.  7.100
libpostproc    56.  6.100 / 56.  6.100
[Parsed_volumedetect_0 @ 0x6000036482c0] n_samples: 19732480
[Parsed_volumedetect_0 @ 0x6000036482c0] mean_volume: -10.5 dB
[Parsed_volumedetect_0 @ 0x6000036482c0] max_volume: -5.0 dB
[Parsed_volumedetect_0 @ 0x6000036482c0] histogram_0db: 184156`

func TestVolumeDetect(t *testing.T) {
	// monkey patching
	patchexecCmdRun := gomonkey.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		return util.ErrOnly(cmd.Stdout.Write([]byte(volumeDetectOutput)))
	})
	defer patchexecCmdRun.Reset()

	// testing
	delta, err := FFmpeg().VolumeDetect("/dev/null")
	assert.Nil(t, err)
	assert.Equal(t, -5.0, delta)
}

func TestVolumeDetectFFmpegFailure(t *testing.T) {
	// monkey patching
	patchexecCmdRun := gomonkey.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		return errors.New("failure")
	})
	defer patchexecCmdRun.Reset()

	// testing
	assert.Error(t, util.ErrOnly(FFmpeg().VolumeDetect("/dev/null")), "failure")
}

func TestVolumeDetectParseFloatFailure(t *testing.T) {
	// monkey patching
	patchexecCmdRun := gomonkey.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		return util.ErrOnly(cmd.Stdout.Write([]byte(volumeDetectOutput)))
	})
	defer patchexecCmdRun.Reset()
	patchstrconvParseFloat := gomonkey.ApplyFunc(strconv.ParseFloat, func(string, int) (float64, error) {
		return 0, errors.New("failure")
	})
	defer patchstrconvParseFloat.Reset()

	// testing
	assert.Error(t, util.ErrOnly(FFmpeg().VolumeDetect("/dev/null")), "failure")
}

func TestVolumeAdd(t *testing.T) {
	// monkey patching
	patchexecCmdRun := gomonkey.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		return nil
	})
	defer patchexecCmdRun.Reset()

	// testing
	assert.Error(t, FFmpeg().VolumeAdd("/dev/null", -1), "rename /dev/null.normalized /dev/null: no such file or directory")
}

func TestVolumeAddNothing(t *testing.T) {
	assert.Nil(t, FFmpeg().VolumeAdd("/dev/null", 0))
}

func TestVolumeAddFFmpegFailure(t *testing.T) {
	// monkey patching
	patchexecCmdRun := gomonkey.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		return errors.New("failure")
	})
	defer patchexecCmdRun.Reset()

	// testing
	assert.Error(t, FFmpeg().VolumeAdd("/dev/null", -1), "failure")
}
