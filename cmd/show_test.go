package cmd

import (
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

func BenchmarkShow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdShow(&testing.T{})
	}
}

func TestCmdShow(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(id3.Open, func() (*id3.Tag, error) {
			return &id3.Tag{}, nil
		}).
		ApplyMethod(reflect.TypeOf(&id3.Tag{}), "AttachedPicture", func() (string, []byte) {
			return "image/jpeg", []byte("some picture data")
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdShow(), "path/to/track")))
}

func TestCmdShowOpenFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(id3.Open, func() (*id3.Tag, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, util.ErrOnly(testExecute(cmdShow(), "path/to/track")), "ko")
}

func TestCmdShowPictureFallback(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(id3.Open, func() (*id3.Tag, error) {
			return &id3.Tag{}, nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdShow(), "path/to/track")))
}
