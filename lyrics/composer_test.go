package lyrics

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

var track = &entity.Track{
	Title:   "Title",
	Artists: []string{"Artist"},
}

func TestSearch(t *testing.T) {
	// monkey patching
	ch := make(chan bool, 1)
	monkey.PatchInstanceMethod(reflect.TypeOf(genius{}), "Search",
		func(genius, *entity.Track, ...context.Context) ([]byte, error) {
			defer close(ch)
			return []byte("glyrics"), nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(genius{}), "Search")
	monkey.PatchInstanceMethod(reflect.TypeOf(lyricsOvh{}), "Search",
		func(lyricsOvh, *entity.Track, ...context.Context) ([]byte, error) {
			<-ch
			return []byte("olyrics"), nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(lyricsOvh{}), "Search")

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Equal(t, []byte("glyrics"), lyrics)
}

func TestSearchFailure(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(genius{}), "Search",
		func(genius, *entity.Track, ...context.Context) ([]byte, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(genius{}), "Search")
	monkey.PatchInstanceMethod(reflect.TypeOf(lyricsOvh{}), "Search",
		func(lyricsOvh, *entity.Track, ...context.Context) ([]byte, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(lyricsOvh{}), "Search")

	// testing
	assert.Error(t, util.ErrOnly(Search(track)), "failure")
}

func TestSearchNotFound(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(genius{}), "Search",
		func(genius, *entity.Track, ...context.Context) ([]byte, error) {
			return nil, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(genius{}), "Search")
	monkey.PatchInstanceMethod(reflect.TypeOf(lyricsOvh{}), "Search",
		func(lyricsOvh, *entity.Track, ...context.Context) ([]byte, error) {
			return nil, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(lyricsOvh{}), "Search")

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Nil(t, lyrics)
}
