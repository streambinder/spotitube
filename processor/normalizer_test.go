package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkNormalizer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestNormalizerDo(&testing.T{})
	}
}

func TestNormalizerDo(t *testing.T) {
	useTempCache(t)
	fakeFFmpeg(t, "-23.45", 0, 0)

	assert.Nil(t, normalizer{}.Do(track))
}

func TestNormalizerDoAboveTarget(t *testing.T) {
	useTempCache(t)
	fakeFFmpeg(t, "-8.0", 0, 0)

	assert.Nil(t, normalizer{}.Do(track))
}

func TestNormalizerDoUnsupported(t *testing.T) {
	assert.NotNil(t, normalizer{}.Do("hello"))
}

func TestNormalizerDoDetectFailure(t *testing.T) {
	useTempCache(t)
	fakeFFmpeg(t, "-23.45", 1, 0)

	// a detection failure skips normalization without failing
	assert.ErrorIs(t, normalizer{}.Do(track), ErrLoudnessSkipped)
}

func TestNormalizerDoNormalizeFailure(t *testing.T) {
	useTempCache(t)
	fakeFFmpeg(t, "-23.45", 0, 1)

	assert.ErrorContains(t, normalizer{}.Do(track), "ko")
}
