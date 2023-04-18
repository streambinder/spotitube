package cmd

import (
	"os/exec"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestYouTubeDlDownload(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(&exec.Cmd{}, "Run", func() error { return nil }).Reset()

	// testing
	assert.Nil(t, YouTubeDl("http://localhost", "fname.txt"))
}
