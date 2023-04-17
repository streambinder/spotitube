package lyrics

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
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
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) { return nil, errors.New("") })
	defer patchosReadFile.Reset()
	patchgeniusSearch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(genius{}), "search",
		func(genius, *entity.Track, ...context.Context) ([]byte, error) {
			defer close(ch)
			return []byte("glyrics"), nil
		})
	defer patchgeniusSearch.Reset()
	patchlyricsOvhSearch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search",
		func(lyricsOvh, *entity.Track, ...context.Context) ([]byte, error) {
			<-ch
			return []byte("olyrics"), nil
		})
	defer patchlyricsOvhSearch.Reset()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Equal(t, "glyrics", lyrics)
}

func TestSearchAlreadyExists(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) { return []byte("lyrics"), nil })
	defer patchosReadFile.Reset()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Equal(t, "lyrics", lyrics)
}

func TestSearchFailure(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) { return nil, errors.New("") })
	defer patchosReadFile.Reset()
	patchgeniusSearch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(genius{}), "search",
		func(genius, *entity.Track, ...context.Context) ([]byte, error) {
			return nil, errors.New("failure")
		})
	defer patchgeniusSearch.Reset()
	patchlyricsOvhSearch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search",
		func(lyricsOvh, *entity.Track, ...context.Context) ([]byte, error) {
			return nil, errors.New("failure")
		})
	defer patchlyricsOvhSearch.Reset()

	// testing
	assert.Error(t, util.ErrOnly(Search(track)), "failure")
}

func TestSearchNotFound(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) { return nil, errors.New("") })
	defer patchosReadFile.Reset()
	patchgeniusSearch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(genius{}), "search",
		func(genius, *entity.Track, ...context.Context) ([]byte, error) {
			return nil, nil
		})
	defer patchgeniusSearch.Reset()
	patchlyricsOvhSearch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search",
		func(lyricsOvh, *entity.Track, ...context.Context) ([]byte, error) {
			return nil, nil
		})
	defer patchlyricsOvhSearch.Reset()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Empty(t, lyrics)
}

func TestSearchCannotCreateDir(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) { return nil, errors.New("") })
	defer patchosReadFile.Reset()
	patchgeniusSearch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(genius{}), "search",
		func(genius, *entity.Track, ...context.Context) ([]byte, error) {
			return []byte("lyrics"), nil
		})
	defer patchgeniusSearch.Reset()
	patchlyricsOvhSearch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search",
		func(lyricsOvh, *entity.Track, ...context.Context) ([]byte, error) {
			return []byte{}, nil
		})
	defer patchlyricsOvhSearch.Reset()
	patchosMkdirAll := gomonkey.ApplyFunc(os.MkdirAll, func(string, fs.FileMode) error { return errors.New("failure") })
	defer patchosMkdirAll.Reset()

	// testing
	assert.Error(t, util.ErrOnly(Search(track)), "failure")
}
