package processor

import (
	"errors"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/entity"
	"github.com/stretchr/testify/assert"
)

var track = &entity.Track{
	ID:       "123",
	Title:    "Title",
	Artists:  []string{"Artist"},
	Album:    "Album",
	Artwork:  entity.Artwork{URL: "http://ima.ge"},
	Duration: 180,
	Number:   1,
	Year:     1970,
}

func BenchmarkProcessor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestProcessorDo(&testing.T{})
	}
}

func TestProcessorDo(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(normalizer{}, "Do")).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(encoder{}, "Do")).Return(nil).Build()

	// testing
	assert.Nil(t, Do(track))
}

func TestProcessorDoFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(normalizer{}, "Do")).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(encoder{}, "Do")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, Do(track), "ko")
}
