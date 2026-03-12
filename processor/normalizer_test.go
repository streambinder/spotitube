package processor

import (
	"errors"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/sys/cmd"
	"github.com/stretchr/testify/assert"
)

func BenchmarkNormalizer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestNormalizerDo(&testing.T{})
	}
}

func TestNormalizerDo(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "VolumeDetect")).Return(float64(1), nil).Build()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "VolumeAdd")).Return(nil).Build()

	// testing
	assert.Nil(t, normalizer{}.Do(track))
}

func TestNormalizerDoReverse(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "VolumeDetect")).Return(float64(-1), nil).Build()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "VolumeAdd")).Return(nil).Build()

	// testing
	assert.Nil(t, normalizer{}.Do(track))
}

func TestNormalizerDoUnsupported(t *testing.T) {
	// testing
	assert.NotNil(t, normalizer{}.Do("hello"))
}

func TestNormalizerDoFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "VolumeDetect")).Return(float64(0), errors.New("ko")).Build()

	// testing
	assert.EqualError(t, normalizer{}.Do(track), "ko")
}

func TestNormalizerDoVolumeAddFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "VolumeDetect")).Return(float64(-1), nil).Build()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "VolumeAdd")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, normalizer{}.Do(track), "ko")
}
