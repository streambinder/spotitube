package provider

import (
	"errors"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys"
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

func BenchmarkProvider(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestSearch(&testing.T{})
	}
}

func TestSearch(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(youTube{}, "search")).Return([]*Match{
		{URL: "url1", Score: 3},
		{URL: "url2", Score: 1},
	}, nil).Build()

	// testing
	matches, err := Search(track)
	assert.Nil(t, err)
	assert.NotEmpty(t, matches)
}

func TestSearchFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(youTube{}, "search")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(Search(track)), "ko")
}
