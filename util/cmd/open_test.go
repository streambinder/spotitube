package cmd

import (
	"os/exec"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func BenchmarkOpen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestOpen(&testing.T{})
	}
}

func TestOpen(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(&exec.Cmd{}, "Start", func() error { return nil }).Reset()

	// testing
	assert.Nil(t, Open("https://davidepucci.it", "linux"))
	assert.Nil(t, Open("https://davidepucci.it", "darwin"))
	assert.Nil(t, Open("https://davidepucci.it", "windows"))
	assert.EqualError(t, Open("https://davidepucci.it", "unknown"), "unsupported platform")
}
