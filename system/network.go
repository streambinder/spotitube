package system

import (
	"io"
	"net/http"
	"os"
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

// Wget performs a GET over a given asset URL writing it on a give filename
func Wget(url, fname string) error {
	if err := os.Remove(fname); FileExists(fname) && err != nil {
		return err
	}

	file, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer file.Close()

	res, err := Client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if _, err := io.Copy(file, res.Body); err != nil {
		return err
	}

	return nil
}
