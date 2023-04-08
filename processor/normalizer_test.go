package processor

import (
	"errors"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
)

func TestNormalizerDo(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(cmd.FFmpegCmd{}), "VolumeDetect",
		func(cmd.FFmpegCmd, string) (float64, error) {
			return 1, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(cmd.FFmpegCmd{}), "VolumeDetect")
	monkey.PatchInstanceMethod(reflect.TypeOf(cmd.FFmpegCmd{}), "VolumeAdd",
		func(cmd.FFmpegCmd, string, float64) error {
			return nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(cmd.FFmpegCmd{}), "VolumeAdd")

	// testing
	assert.Nil(t, normalizer{}.Do(track))
}

func TestNormalizerDoFailure(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(cmd.FFmpegCmd{}), "VolumeDetect",
		func(cmd.FFmpegCmd, string) (float64, error) {
			return 0, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(cmd.FFmpegCmd{}), "VolumeDetect")

	// testing
	assert.Error(t, normalizer{}.Do(track), "failure")
}
