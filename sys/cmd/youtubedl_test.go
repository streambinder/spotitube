package cmd

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func BenchmarkYouTubeDl(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestYouTubeDlDownload(&testing.T{})
	}
}

func TestYouTubeDlDownload(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(&exec.Cmd{}, "Run", func() error { return nil }).Reset()

	// testing
	assert.Nil(t, YouTubeDl("http://localhost", "fname.txt"))
}

func TestYouTubeDlDownloadFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(&exec.Cmd{}, "Run", func() error { return errors.New("ko") }).Reset()

	// testing
	assert.Error(t, YouTubeDl("http://localhost", "fname.txt"))
}
