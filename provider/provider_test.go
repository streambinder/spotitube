package provider

import (
	"errors"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

var track = &entity.Track{
	ID:         "123",
	Title:      "Title",
	Artists:    []string{"Artist"},
	Album:      "Album",
	ArtworkURL: "http://ima.ge",
	Duration:   180,
	Number:     1,
	Year:       "1970",
}

func TestSearch(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(youTube{}), "Search",
		func(provider youTube, track *entity.Track) ([]*Match, error) {
			return []*Match{
				{URL: "url1", Score: 3},
				{URL: "url2", Score: 1},
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(youTube{}), "Search")

	// testing
	matches, err := Search(track)
	assert.Nil(t, err)
	assert.NotEmpty(t, matches)
}

func TestSearchFailure(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(youTube{}), "Search",
		func(provider youTube, track *entity.Track) ([]*Match, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(youTube{}), "Search")

	// testing
	assert.Error(t, util.ErrOnly(Search(track)), "failure")
}
