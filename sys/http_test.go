package sys

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func BenchmarkHTTP(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestSleepUntilRetry(&testing.T{})
	}
}

func TestSleepUntilRetry(t *testing.T) {
	var duration time.Duration
	sleepFn = func(d time.Duration) { duration = d }
	defer func() { sleepFn = time.Sleep }()

	SleepUntilRetry(http.Header{"Retry-After": []string{"10"}})
	assert.Equal(t, duration, 10*time.Second)
}

func TestSleepUntilRetryNoHeader(t *testing.T) {
	var duration time.Duration
	sleepFn = func(d time.Duration) { duration = d }
	defer func() { sleepFn = time.Sleep }()

	SleepUntilRetry(http.Header{})
	assert.Equal(t, duration, defaultRetryWait)
}

func TestSleepUntilRetryInvalidHeader(t *testing.T) {
	var duration time.Duration
	sleepFn = func(d time.Duration) { duration = d }
	defer func() { sleepFn = time.Sleep }()

	SleepUntilRetry(http.Header{"Retry-After": []string{"not-a-number"}})
	assert.Equal(t, duration, defaultRetryWait)
}
