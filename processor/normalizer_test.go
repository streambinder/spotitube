package processor

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
)

func BenchmarkNormalizer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestNormalizerDo(&testing.T{})
	}
}

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
	assert.Nil(t, normalizer{}.Do(track))
}

func TestNormalizerDoUnsupported(t *testing.T) {
	// testing
	assert.NotNil(t, normalizer{}.Do("hello"))
}

func TestNormalizerDoFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(cmd.FFmpegCmd{}, "VolumeDetect", func() (float64, error) {
		return 0, errors.New("ko")
	}).Reset()

	// testing
	assert.Error(t, normalizer{}.Do(track), "ko")
}
