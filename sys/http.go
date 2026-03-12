package sys

import (
	"net/http"
	"strconv"
	"time"
)

const defaultRetryWait = 5 * time.Second

var sleepFn = time.Sleep

func SleepUntilRetry(headers http.Header) {
	waitDuration := defaultRetryWait
	if header := headers.Get("Retry-After"); header != "" {
		if seconds, err := strconv.ParseInt(header, 10, 32); err == nil {
			waitDuration = time.Duration(seconds) * time.Second
		}
	}
	sleepFn(waitDuration)
}
