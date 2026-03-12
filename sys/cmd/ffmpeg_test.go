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

func BenchmarkFFmpeg(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestVolumeDetect(&testing.T{})
		TestVolumeAdd(&testing.T{})
	}
}

func TestVolumeDetect(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).To(func(cmd *exec.Cmd) error {
		return sys.ErrOnly(cmd.Stdout.Write([]byte(volumeDetectOutput)))
	}).Build()

	// testing
	delta, err := FFmpeg().VolumeDetect("/dev/null")
	assert.Nil(t, err)
	assert.Equal(t, -5.0, delta)
}

func TestVolumeDetectFFmpegFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).Return(errors.New("ko")).Build()

	// testing
	assert.Error(t, sys.ErrOnly(FFmpeg().VolumeDetect("/dev/null")))
}

func TestVolumeDetectParseFloatFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).To(func(cmd *exec.Cmd) error {
		return sys.ErrOnly(cmd.Stdout.Write([]byte(volumeDetectOutput)))
	}).Build()
	mockey.Mock(strconv.ParseFloat).Return(0.0, errors.New("ko")).Build()

	// testing
	assert.Error(t, sys.ErrOnly(FFmpeg().VolumeDetect("/dev/null")))
}

func TestVolumeDetectNoMatch(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).To(func(cmd *exec.Cmd) error {
		return sys.ErrOnly(cmd.Stdout.Write([]byte("no volume info here")))
	}).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(FFmpeg().VolumeDetect("/dev/null")), "cannot parse max_volume for given track")
}

func TestVolumeAdd(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).Return(nil).Build()
	mockey.Mock(os.Rename).Return(nil).Build()

	// testing
	assert.Nil(t, FFmpeg().VolumeAdd("/dev/null", -1))
}

func TestVolumeAddRenameFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).Return(nil).Build()
	mockey.Mock(os.Rename).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, FFmpeg().VolumeAdd("/dev/null", -1), "ko")
}

func TestVolumeAddNothing(t *testing.T) {
	assert.Nil(t, FFmpeg().VolumeAdd("/dev/null", 0))
}

func TestVolumeAddFFmpegFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).Return(errors.New("ko")).Build()

	// testing
	assert.Error(t, FFmpeg().VolumeAdd("/dev/null", -1))
}
