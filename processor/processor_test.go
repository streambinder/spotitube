package processor

import (
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
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

func TestProcessorDo(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyPrivateMethod(reflect.TypeOf(normalizer{}), "Do", func() error {
			return nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(encoder{}), "Do", func() error {
			return nil
		}).
		Reset()

	// testing
	assert.Nil(t, Do(track))
}

func TestProcessorDoFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyPrivateMethod(reflect.TypeOf(normalizer{}), "Do", func() error {
			return nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(encoder{}), "Do", func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, Do(track), "ko")
}
