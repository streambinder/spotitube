package cmd

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
)

func BenchmarkYouTubeDl(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestYouTubeDlDownload(&testing.T{})
	}
}

func TestYouTubeDlDownload(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).Return(nil).Build()

	// testing
	assert.Nil(t, YouTubeDl("http://localhost", "fname.txt"))
}

func TestYouTubeDlDownloadFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&exec.Cmd{}, "Run")).Return(errors.New("ko")).Build()

	// testing
	assert.Error(t, YouTubeDl("http://localhost", "fname.txt"))
}
