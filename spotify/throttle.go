package spotify

import (
	"errors"
	"strings"
	"time"
)

var (
	errThrottle = errors.New("rate limited")
)

func wrapThrottling(err error) (wrapErr error) {
	if err == nil {
		return
	} else if strings.Contains(err.Error(), "rate limit") {
		time.Sleep(5 * time.Second)
		return errThrottle
	}
	return err
}
