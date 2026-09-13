package provider

import (
	"errors"
	"testing"

	"github.com/arunsworld/nursery"
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
	mockey.Mock(mockey.GetMethod(qobuz{}, "search")).Return([]*Match{
		{URL: "url3", Score: 100},
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
	mockey.Mock(mockey.GetMethod(qobuz{}, "search")).Return(nil, errors.New("ko")).Build()

	// all providers failed → propagate as error
	_, err := Search(track)
	assert.EqualError(t, err, "all providers failed")
}

func TestSearchPartialFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(youTube{}, "search")).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(qobuz{}, "search")).Return([]*Match{{URL: "url1", Score: 100}}, nil).Build()

	// one provider succeeded → return its matches, no error
	matches, err := Search(track)
	assert.Nil(t, err)
	assert.NotEmpty(t, matches)
}

func TestSearchNurseryFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(nursery.RunConcurrently).Return(errors.New("nursery ko")).Build()

	// testing
	_, err := Search(track)
	assert.EqualError(t, err, "nursery ko")
}
