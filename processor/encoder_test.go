package processor

import (
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/bogem/id3v2/v2"
	"github.com/stretchr/testify/assert"
)

func TestEncoderDo(t *testing.T) {
	// monkey patching
	patchid3v2Open := gomonkey.ApplyFunc(id3v2.Open, func(string, id3v2.Options) (*id3v2.Tag, error) {
		return id3v2.NewEmptyTag(), nil
	})
	defer patchid3v2Open.Reset()
	patchid3v2TagSave := gomonkey.ApplyMethod(reflect.TypeOf(&id3v2.Tag{}), "Save",
		func(*id3v2.Tag) error { return nil })
	defer patchid3v2TagSave.Reset()

	// testing
	assert.Nil(t, encoder{}.do(track))
}

func TestEncoderDoOpenFailure(t *testing.T) {
	// monkey patching
	patchid3v2Open := gomonkey.ApplyFunc(id3v2.Open, func(string, id3v2.Options) (*id3v2.Tag, error) {
		return nil, errors.New("failure")
	})
	defer patchid3v2Open.Reset()

	// testing
	assert.Error(t, encoder{}.do(track), "failure")
}
