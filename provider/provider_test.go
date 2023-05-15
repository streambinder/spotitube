package provider

import (
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
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
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(youTube{}), "search", func() ([]*Match, error) {
		return []*Match{
			{URL: "url1", Score: 3},
			{URL: "url2", Score: 1},
		}, nil
	}).Reset()

	// testing
	matches, err := Search(track)
	assert.Nil(t, err)
	assert.NotEmpty(t, matches)
}

func TestSearchFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(youTube{}), "search", func() ([]*Match, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.Error(t, util.ErrOnly(Search(track)), "ko")
}
