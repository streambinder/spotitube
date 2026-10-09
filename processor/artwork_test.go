package processor

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkArtwork(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestArtworkDo(&testing.T{})
	}
}

func TestArtworkDo(t *testing.T) {
	var source bytes.Buffer
	if err := jpeg.Encode(&source, image.NewRGBA(image.Rect(0, 0, 600, 400)), nil); err != nil {
		t.Fatal(err)
	}
	data := source.Bytes()

	assert.Nil(t, Artwork{}.Do(&data))

	// the artwork has been resized to a 300px-wide JPEG
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	assert.Nil(t, err)
	assert.Equal(t, "jpeg", format)
	assert.Equal(t, 300, config.Width)
}

func TestArtworkDoUnsupported(t *testing.T) {
	assert.NotNil(t, Artwork{}.Do(track))
}

func TestArtworkDoDecodeFailure(t *testing.T) {
	data := []byte("this is not an image")

	assert.ErrorContains(t, Artwork{}.Do(&data), "unknown format")
}
