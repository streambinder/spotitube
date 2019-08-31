package system

import (
	"net/http"
	"time"
)

const (
	// HTTPTimeout is the default timeout used for HTTP calls
	HTTPTimeout = 3 // second(s)
)

var (
	// Client is a generic HTTP Client usable widely
	Client = http.Client{Timeout: time.Second * HTTPTimeout}
)

// IsOnline checks internet connection coherently returning a boolean value
func IsOnline() bool {
	r, _ := http.NewRequest("GET", "http://clients3.google.com/generate_204", nil)
	_, err := Client.Do(r)
	return err == nil
}
