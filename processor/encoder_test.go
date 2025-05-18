package processor

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/id3v2-sylt"
	"github.com/stretchr/testify/assert"
)

func BenchmarkEncoder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestEncoderDo(&testing.T{})
	}
}

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
	assert.Nil(t, encoder{}.Do(track))
}

func TestEncoderDoUnsupported(t *testing.T) {
	// testing
	assert.NotNil(t, encoder{}.Do("hello"))
}

func TestEncoderDoOpenFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, encoder{}.Do(track), "ko")
}
