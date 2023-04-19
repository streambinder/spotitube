package processor

import (
	"errors"
	"image"
	"image/jpeg"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestArtworkDo(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(image.Decode, func() (image.Image, string, error) {
			return image.NewRGBA(image.Rectangle{image.Pt(0, 0), image.Pt(0, 0)}), "", nil
		}).
		ApplyFunc(jpeg.Encode, func() error {
			return nil
		}).
		Reset()

	// testing
	assert.Nil(t, artwork{}.do(&[]byte{}))
}

func TestArtworkDoUnsupported(t *testing.T) {
	// testing
	assert.NotNil(t, artwork{}.do(track))
}

func TestEncoderDoDecodeFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(image.Decode, func() (image.Image, string, error) {
		return image.NewRGBA(image.Rectangle{image.Pt(0, 0), image.Pt(0, 0)}), "", errors.New("ko")
	}).Reset()

	// testing
	assert.Error(t, artwork{}.do(&[]byte{}), "ko")
}

func TestEncoderDoEncodeFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(image.Decode, func() (image.Image, string, error) {
			return image.NewRGBA(image.Rectangle{image.Pt(0, 0), image.Pt(0, 0)}), "", nil
		}).
		ApplyFunc(jpeg.Encode, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, artwork{}.do(&[]byte{}), "ko")
}
