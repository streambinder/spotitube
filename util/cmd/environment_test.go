package cmd

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func BenchmarkEnvironment(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestValidateEnvironment(&testing.T{})
	}
}

func TestValidateEnvironment(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(exec.LookPath, func() (string, error) { return "", nil }).Reset()

	// testing
	assert.Nil(t, ValidateEnvironment())
}

func TestValidateEnvironmentNoFFmpeg(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(exec.LookPath, func(file string) (string, error) {
		if file == "ffmpeg" {
			return "", fmt.Errorf("no ffmpeg")
		}
		return "", nil
	}).Reset()

	// testing
	assert.Error(t, ValidateEnvironment(), "command \"ffmpeg\" not found in PATH")
}
