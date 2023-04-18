package processor

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/bogem/id3v2/v2"
	"github.com/stretchr/testify/assert"
)

func TestEncoderDo(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
			return id3v2.NewEmptyTag(), nil
		}).
		ApplyMethod(&id3v2.Tag{}, "Save", func() error {
			return nil
		}).
		Reset()

	// testing
	assert.Nil(t, encoder{}.do(track))
}

func TestEncoderDoOpenFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.Error(t, encoder{}.do(track), "ko")
}
