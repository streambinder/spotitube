package anchor

import (
	"io"
	"os"
	"testing"

	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

func BenchmarkWindow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestWindow(&testing.T{})
	}
}

func TestWindow(t *testing.T) {
	stdout := os.Stdout
	defer func() {
		os.Stdout = stdout
	}()
	reader, writer, err := os.Pipe()
	assert.Nil(t, err)
	os.Stdout = writer

	stdin := os.Stdin
	defer func() {
		os.Stdin = stdin
	}()
	stdinFile, err := os.CreateTemp(util.CacheDirectory(), "test")
	assert.Nil(t, err)
	defer os.Remove(stdinFile.Name())
	assert.Nil(t, util.ErrOnly(stdinFile.Write([]byte("input\n"))))
	os.Stdin = stdinFile

	var (
		window = Window(Normal)
		lot    = window.Lot("lot")
	)
	lot.Printf("lot text 1")
	window.Printf("default text 1")
	window.AnchorPrintf("anchor text")
	window.Printf("default text 2")
	window.Lot("lot").Printf("lot text 2")
	window.shift(-2)
	lot.Wipe()
	lot.Close("closure")
	window.Reads("prompt:")
	assert.Nil(t, writer.Close())

	output, err := io.ReadAll(reader)
	assert.Nil(t, err)
	assert.Contains(t, string(output), "lot text 1")
	assert.Contains(t, string(output), "default text 1")
	assert.Contains(t, string(output), "anchor text")
	assert.Contains(t, string(output), "default text 2")
	assert.Contains(t, string(output), "lot text 2")
	assert.Contains(t, string(output), "closure")
	assert.Contains(t, string(output), "prompt")
}
