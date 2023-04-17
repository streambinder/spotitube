package processor

import (
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
)

func TestNormalizerDo(t *testing.T) {
	// monkey patching
	patchcmdFFmpegCmdVolumeDetect := gomonkey.ApplyMethod(reflect.TypeOf(cmd.FFmpegCmd{}), "VolumeDetect",
		func(cmd.FFmpegCmd, string) (float64, error) {
			return 1, nil
		})
	defer patchcmdFFmpegCmdVolumeDetect.Reset()
	patchcmdFFmpegCmdVolumeAdd := gomonkey.ApplyMethod(reflect.TypeOf(cmd.FFmpegCmd{}), "VolumeAdd",
		func(cmd.FFmpegCmd, string, float64) error {
			return nil
		})
	defer patchcmdFFmpegCmdVolumeAdd.Reset()

	// testing
	assert.Nil(t, normalizer{}.do(track))
}

func TestNormalizerDoFailure(t *testing.T) {
	// monkey patching
	patchcmdFFmpegCmdVolumeDetect := gomonkey.ApplyMethod(reflect.TypeOf(cmd.FFmpegCmd{}), "VolumeDetect",
		func(cmd.FFmpegCmd, string) (float64, error) {
			return 0, errors.New("failure")
		})
	defer patchcmdFFmpegCmdVolumeDetect.Reset()

	// testing
	assert.Error(t, normalizer{}.do(track), "failure")
}
