package processor

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
)

func TestNormalizerDo(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyMethod(cmd.FFmpegCmd{}, "VolumeDetect", func() (float64, error) {
			return 1, nil
		}).
		ApplyMethod(cmd.FFmpegCmd{}, "VolumeAdd", func() error {
			return nil
		}).
		Reset()

	// testing
	assert.Nil(t, normalizer{}.do(track))
}

func TestNormalizerDoFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(cmd.FFmpegCmd{}, "VolumeDetect", func() (float64, error) {
		return 0, errors.New("ko")
	}).Reset()

	// testing
	assert.Error(t, normalizer{}.do(track), "ko")
}
