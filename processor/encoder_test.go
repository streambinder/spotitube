package processor

import (
	"errors"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/bogem/id3v2/v2"
	"github.com/stretchr/testify/assert"
)

func TestEncoderDo(t *testing.T) {
	// monkey patching
	monkey.Patch(id3v2.Open, func(string, id3v2.Options) (*id3v2.Tag, error) {
		return id3v2.NewEmptyTag(), nil
	})
	defer monkey.Unpatch(id3v2.Open)
	monkey.PatchInstanceMethod(reflect.TypeOf(&id3v2.Tag{}), "Save",
		func(*id3v2.Tag) error { return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&id3v2.Tag{}), "Save")

	// testing
	assert.Nil(t, encoder{}.Do(track))
}

func TestEncoderDoOpenFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(id3v2.Open, func(string, id3v2.Options) (*id3v2.Tag, error) {
		return nil, errors.New("failure")
	})
	defer monkey.Unpatch(id3v2.Open)

	// testing
	assert.Error(t, encoder{}.Do(track), "failure")
}
