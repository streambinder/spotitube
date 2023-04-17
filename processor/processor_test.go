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
	Year:     "1970",
}

func TestProcessorDo(t *testing.T) {
	// monkey patching
	patchnormalizerDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(normalizer{}), "do",
		func(normalizer, *entity.Track) error {
			return nil
		})
	defer patchnormalizerDo.Reset()
	patchencoderDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(encoder{}), "do",
		func(encoder, *entity.Track) error {
			return nil
		})
	defer patchencoderDo.Reset()

	// testing
	assert.Nil(t, Do(track))
}

func TestProcessorDoFailure(t *testing.T) {
	// monkey patching
	patchnormalizerDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(normalizer{}), "do",
		func(normalizer, *entity.Track) error {
			return nil
		})
	defer patchnormalizerDo.Reset()
	patchencoderDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(encoder{}), "do",
		func(encoder, *entity.Track) error {
			return errors.New("failure")
		})
	defer patchencoderDo.Reset()

	// testing
	assert.Error(t, Do(track), "failure")
}
