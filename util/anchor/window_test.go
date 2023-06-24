package anchor

import (
	"io"
	"os"
	"testing"

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
	assert.Nil(t, writer.Close())

	output, err := io.ReadAll(reader)
	assert.Nil(t, err)
	assert.Contains(t, string(output), "lot text 1")
	assert.Contains(t, string(output), "default text 1")
	assert.Contains(t, string(output), "anchor text")
	assert.Contains(t, string(output), "default text 2")
	assert.Contains(t, string(output), "lot text 2")
	assert.Contains(t, string(output), "closure")
}
