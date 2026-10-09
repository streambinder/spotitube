package spotify

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

// scriptTransport swaps http.DefaultTransport for a scripted handler for the
// duration of a test. Every HTTP boundary of the code under test bottoms out
// in http.DefaultTransport: the Spotify Web API through http.DefaultClient,
// and the OAuth2 token endpoint through the authenticator's client. The
// handler receives each request and returns the status code and JSON body to
// respond with, keeping the suite hermetic.
func scriptTransport(t *testing.T, handler func(*http.Request) (int, string)) {
	t.Helper()
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		status, body := handler(request)
		return &http.Response{
			Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
			StatusCode:    status,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        http.Header{"Content-Type": []string{"application/json"}},
			Body:          io.NopCloser(strings.NewReader(body)),
			ContentLength: int64(len(body)),
			Request:       request,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
}

// apiError returns a Spotify-shaped error payload whose message the client
// surfaces verbatim as the returned error.
func apiError(status int, message string) (int, string) {
	return status, fmt.Sprintf(`{"error":{"status":%d,"message":%q}}`, status, message)
}

const (
	// meJSON is the payload of GET /v1/me for the test user.
	meJSON = `{"id":"alice","display_name":"Alice"}`

	// tokenJSON is the payload of the OAuth2 token endpoint.
	tokenJSON = `{"access_token":"access","token_type":"Bearer",` +
		`"refresh_token":"refresh","expires_in":3600}`

	// trackJSON mirrors the fullTrack fixture as a GET /v1/tracks payload.
	trackJSON = `{"id":"123","name":"Title","artists":[{"name":"Artist"}],` +
		`"duration_ms":180000,"track_number":1,` +
		`"album":{"name":"Album","release_date":"1970","images":[{"url":"http://ima.ge"}]}}`

	// simpleTrackJSON mirrors fullTrack.SimpleTrack as an album track item.
	simpleTrackJSON = `{"id":"123","name":"Title","artists":[{"name":"Artist"}],` +
		`"duration_ms":180000,"track_number":1}`
)

// scriptAuthTransport scripts the two hosts the authentication flow talks
// to: the OAuth2 token endpoint always issues a fresh token, while requests
// to the Web API are delegated to the given handler.
func scriptAuthTransport(t *testing.T, api func(*http.Request) (int, string)) {
	t.Helper()
	scriptTransport(t, func(request *http.Request) (int, string) {
		if request.URL.Host == "accounts.spotify.com" {
			return http.StatusOK, tokenJSON
		}
		return api(request)
	})
}
