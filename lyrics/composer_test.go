package lyrics

import (
	"errors"
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
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) { return nil, errors.New("") }).
		ApplyPrivateMethod(reflect.TypeOf(genius{}), "search", func() ([]byte, error) {
			close(ch)
			return []byte("glyrics"), nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search", func() ([]byte, error) {
			<-ch
			return []byte("olyrics"), nil
		}).
		Reset()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Equal(t, "glyrics", lyrics)
}

func TestSearchAlreadyExists(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(os.ReadFile, func() ([]byte, error) {
		return []byte("lyrics"), nil
	}).Reset()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Equal(t, "lyrics", lyrics)
}

func TestSearchFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("")
		}).
		ApplyPrivateMethod(reflect.TypeOf(genius{}), "search", func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search", func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, util.ErrOnly(Search(track)), "ko")
}

func TestSearchNotFound(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("")
		}).
		ApplyPrivateMethod(reflect.TypeOf(genius{}), "search", func() ([]byte, error) {
			return nil, nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search", func() ([]byte, error) {
			return nil, nil
		}).
		Reset()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Empty(t, lyrics)
}

func TestSearchCannotCreateDir(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("")
		}).
		ApplyPrivateMethod(reflect.TypeOf(genius{}), "search", func() ([]byte, error) {
			return []byte("lyrics"), nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search", func() ([]byte, error) {
			return []byte{}, nil
		}).
		ApplyFunc(os.MkdirAll, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, util.ErrOnly(Search(track)), "ko")
}
