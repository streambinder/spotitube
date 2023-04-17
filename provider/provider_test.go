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
	Year:     "1970",
}

func TestSearch(t *testing.T) {
	// monkey patching
	patchyouTubeSearch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(youTube{}), "search",
		func(youTube, *entity.Track) ([]*Match, error) {
			return []*Match{
				{URL: "url1", Score: 3},
				{URL: "url2", Score: 1},
			}, nil
		})
	defer patchyouTubeSearch.Reset()

	// testing
	matches, err := Search(track)
	assert.Nil(t, err)
	assert.NotEmpty(t, matches)
}

func TestSearchFailure(t *testing.T) {
	// monkey patching
	patchyouTubeSearch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(youTube{}), "search",
		func(youTube, *entity.Track) ([]*Match, error) {
			return nil, errors.New("failure")
		})
	defer patchyouTubeSearch.Reset()

	// testing
	assert.Error(t, util.ErrOnly(Search(track)), "failure")
}
