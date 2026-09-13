package processor

import (
	"errors"
	"image"
	"image/jpeg"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
)

func BenchmarkArtwork(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestArtworkDo(&testing.T{})
	}
}

func TestArtworkDo(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(image.Decode).Return(
		image.NewRGBA(image.Rectangle{image.Pt(0, 0), image.Pt(0, 0)}), "", nil,
	).Build()
	mockey.Mock(jpeg.Encode).Return(nil).Build()

	// testing
	assert.Nil(t, Artwork{}.Do(&[]byte{}))
}

func TestArtworkDoUnsupported(t *testing.T) {
	// testing
	assert.NotNil(t, Artwork{}.Do(track))
}

func TestArtworkDoDecodeFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(image.Decode).Return(
		image.NewRGBA(image.Rectangle{image.Pt(0, 0), image.Pt(0, 0)}), "", errors.New("ko"),
	).Build()

	// testing
	assert.EqualError(t, Artwork{}.Do(&[]byte{}), "ko")
}

func TestArtworkDoEncodeFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(image.Decode).Return(
		image.NewRGBA(image.Rectangle{image.Pt(0, 0), image.Pt(0, 0)}), "", nil,
	).Build()
	mockey.Mock(jpeg.Encode).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, Artwork{}.Do(&[]byte{}), "ko")
}
