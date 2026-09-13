package cmd

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
)

func BenchmarkEnvironment(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestValidateEnvironment(&testing.T{})
	}
}

func TestValidateEnvironment(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(exec.LookPath).Return("", nil).Build()

	// testing
	assert.Nil(t, ValidateEnvironment())
}

func TestValidateEnvironmentNoFFmpeg(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(exec.LookPath).To(func(file string) (string, error) {
		if file == "ffmpeg" {
			return "", fmt.Errorf("no ffmpeg")
		}
		return "", nil
	}).Build()

	// testing
	assert.Error(t, ValidateEnvironment(), "command \"ffmpeg\" not found in PATH")
}

func TestValidateEnvironmentNoYtDlp(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(exec.LookPath).To(func(file string) (string, error) {
		if file == "yt-dlp" {
			return "", fmt.Errorf("no yt-dlp")
		}
		return "", nil
	}).Build()

	// testing
	assert.Error(t, ValidateEnvironment(), "command \"yt-dlp\" not found in PATH")
}
