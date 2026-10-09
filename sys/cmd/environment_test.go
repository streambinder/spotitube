package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkEnvironment(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestValidateEnvironment(&testing.T{})
	}
}

func TestValidateEnvironment(t *testing.T) {
	installFakeBinaries(t, map[string]string{
		ffmpegName: "exit 0\n",
		ytdlpName:  "exit 0\n",
	})

	assert.Nil(t, ValidateEnvironment())
}

func TestValidateEnvironmentNoFFmpeg(t *testing.T) {
	installFakeBinaries(t, map[string]string{
		ytdlpName: "exit 0\n",
	})

	assert.EqualError(t, ValidateEnvironment(), "command \"ffmpeg\" not found in PATH")
}

func TestValidateEnvironmentNoYtDlp(t *testing.T) {
	installFakeBinaries(t, map[string]string{
		ffmpegName: "exit 0\n",
	})

	assert.EqualError(t, ValidateEnvironment(), "command \"yt-dlp\" not found in PATH")
}
