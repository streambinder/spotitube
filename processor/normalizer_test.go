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
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "LoudnessDetect")).Return(cmd.Loudness{Integrated: -23.45}, nil).Build()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "LoudnessNormalize")).Return(nil).Build()

	// testing
	assert.Nil(t, normalizer{}.Do(track))
}

func TestNormalizerDoAboveTarget(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "LoudnessDetect")).Return(cmd.Loudness{Integrated: -8.0}, nil).Build()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "LoudnessNormalize")).Return(nil).Build()

	// testing
	assert.Nil(t, normalizer{}.Do(track))
}

func TestNormalizerDoUnsupported(t *testing.T) {
	// testing
	assert.NotNil(t, normalizer{}.Do("hello"))
}

func TestNormalizerDoDetectFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "LoudnessDetect")).Return(cmd.Loudness{}, errors.New("ko")).Build()

	// testing: a detection failure skips normalization without failing
	assert.ErrorIs(t, normalizer{}.Do(track), ErrLoudnessSkipped)
}

func TestNormalizerDoNormalizeFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "LoudnessDetect")).Return(cmd.Loudness{Integrated: -23.45}, nil).Build()
	mockey.Mock(mockey.GetMethod(cmd.FFmpegCmd{}, "LoudnessNormalize")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, normalizer{}.Do(track), "ko")
}
