package sys

import (
	"net/http"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func BenchmarkHTTP(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestSleepUntilRetry(&testing.T{})
	}
}

func TestSleepUntilRetry(t *testing.T) {
	var (
		headers = http.Header{
			"Retry-After": []string{"10"},
		}
		duration time.Duration
	)

	// monkey patching
	defer gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {
		duration = d
	}).Reset()

	// testing
	SleepUntilRetry(headers)
	assert.Equal(t, duration, 10*time.Second)
}

func TestSleepUntilRetryNoHeader(t *testing.T) {
	var duration time.Duration

	// monkey patching
	defer gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {
		duration = d
	}).Reset()

	// testing
	SleepUntilRetry(http.Header{})
	assert.Equal(t, duration, defaultRetryWait)
}
