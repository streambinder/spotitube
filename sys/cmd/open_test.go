package cmd

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
)

func BenchmarkOpen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestOpen(&testing.T{})
	}
}

func TestOpen(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Start")).Return(nil).Build()

	// testing
	assert.Nil(t, Open("https://davidepucci.it", "linux"))
	assert.Nil(t, Open("https://davidepucci.it", "darwin"))
	assert.Nil(t, Open("https://davidepucci.it", "windows"))
	assert.EqualError(t, Open("https://davidepucci.it", "unknown"), "unsupported platform")
}

func TestOpenStartFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Start")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, Open("https://davidepucci.it", "linux"), "ko")
}
