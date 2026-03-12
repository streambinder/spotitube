package processor

import (
	"errors"
	"testing"

	"github.com/bytedance/mockey"
	id3v2 "github.com/streambinder/id3v2-sylt"
	"github.com/stretchr/testify/assert"
)

func BenchmarkEncoder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestEncoderDo(&testing.T{})
	}
}

func TestEncoderDo(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(id3v2.NewEmptyTag(), nil).Build()
	mockey.Mock(mockey.GetMethod(&id3v2.Tag{}, "Save")).Return(nil).Build()

	// testing
	assert.Nil(t, encoder{}.Do(track))
}

func TestEncoderDoUnsupported(t *testing.T) {
	// testing
	assert.NotNil(t, encoder{}.Do("hello"))
}

func TestEncoderDoOpenFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, encoder{}.Do(track), "ko")
}

func TestEncoderDoSaveFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(id3v2.NewEmptyTag(), nil).Build()
	mockey.Mock(mockey.GetMethod(&id3v2.Tag{}, "Save")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, encoder{}.Do(track), "ko")
}
