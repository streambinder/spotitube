package spotify

import (
	"strings"
	"time"
)

type errorType int

const (
	errorStrict = iota
	errorRelaxed
	_
	throttleDelay = 5 * time.Second
)

func (c *Client) handleError(err error) errorType {
	if strings.Contains(err.Error(), "rate limit") {
		c.throttle()
		return errorRelaxed
	}

	return errorStrict
}

func (c *Client) throttle() {
	time.Sleep(throttleDelay)
}
