package sys

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkTime(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestColonMinutesToMillis(&testing.T{})
		TestMillisToColonMinutes(&testing.T{})
	}
}

func TestColonMinutesToMillis(t *testing.T) {
	ms, err := ColonMinutesToMillis("00:15.55")
	assert.NoError(t, err)
	assert.Equal(t, uint32(15550), ms)
}

func TestColonMinutesToMillisMinutesFailure(t *testing.T) {
	assert.Error(t, ErrOnly(ColonMinutesToMillis("X:15.55")))
}

func TestColonMinutesToMillisInvalidFormatFailure(t *testing.T) {
	assert.Error(t, ErrOnly(ColonMinutesToMillis("15")))
}

func TestColonMinutesToMillisSecondsMissingFailure(t *testing.T) {
	assert.Error(t, ErrOnly(ColonMinutesToMillis("00:55")))
}

func TestColonMinutesToMillisSecondsFailure(t *testing.T) {
	assert.Error(t, ErrOnly(ColonMinutesToMillis("00:X.55")))
}

func TestColonMinutesToMillisHundrethsFailure(t *testing.T) {
	assert.Error(t, ErrOnly(ColonMinutesToMillis("00:15.X")))
}

func TestMillisToColonMinutes(t *testing.T) {
	assert.Equal(t, "00:15.55", MillisToColonMinutes(15550))
}
