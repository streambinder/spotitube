package provider

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/streambinder/spotitube/entity"
	"github.com/stretchr/testify/assert"
)

var track = &entity.Track{
	ID:       "123",
	Title:    "Title",
	Artists:  []string{"Artist"},
	Album:    "Album",
	Artwork:  entity.Artwork{URL: "http://ima.ge"},
	Duration: 180,
	Number:   1,
	Year:     1970,
}

// roundTripFunc adapts a function to http.RoundTripper, so tests can
// script every HTTP response the providers receive without any network.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

// stubHTTP routes all provider HTTP traffic through the given handler:
// the shared default transport (youtube) and the qobuz client transport
// are both replaced for the duration of the test.
func stubHTTP(t *testing.T, handler func(*http.Request) (*http.Response, error)) {
	t.Helper()
	previousDefault := http.DefaultTransport
	previousQobuz := qobuzHTTPClient.Transport
	http.DefaultTransport = roundTripFunc(handler)
	qobuzHTTPClient.Transport = roundTripFunc(handler)
	t.Cleanup(func() {
		http.DefaultTransport = previousDefault
		qobuzHTTPClient.Transport = previousQobuz
	})
}

func httpResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// errReader fails every read, to exercise body reading failures.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }
func (r errReader) Close() error             { return nil }

func BenchmarkProvider(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestSearch(&testing.T{})
	}
}

func TestSearch(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		switch request.URL.Host {
		case "www.youtube.com":
			return httpResponse(200, youtubeResultPage()), nil
		default:
			return qobuzFlow(request, qobuzScript{})
		}
	})

	// testing
	matches, err := Search(track)
	assert.Nil(t, err)
	assert.NotEmpty(t, matches)
}

func TestSearchFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("ko")
	})

	// all providers failed → propagate as error
	_, err := Search(track)
	assert.EqualError(t, err, "all providers failed")
}

func TestSearchPartialFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		if request.URL.Host == "www.youtube.com" {
			return nil, errors.New("ko")
		}
		return qobuzFlow(request, qobuzScript{})
	})

	// one provider succeeded → return its matches, no error
	matches, err := Search(track)
	assert.Nil(t, err)
	assert.NotEmpty(t, matches)
}
