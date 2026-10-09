package lyrics

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// roundTripFunc adapts a function to http.RoundTripper, so tests can
// script every HTTP response the composers receive without any network.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

// stubTransport routes all composer HTTP traffic through the given
// handler for the duration of the test: both composers bottom out in
// http.DefaultClient, whose transport is the default one.
func stubTransport(t *testing.T, handler func(*http.Request) (*http.Response, error)) {
	t.Helper()
	stubRoundTripper(t, roundTripFunc(handler))
}

// stubRoundTripper installs the given transport as the default one for
// the duration of the test.
func stubRoundTripper(t *testing.T, transport http.RoundTripper) {
	t.Helper()
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previous })
}

func httpResponse(status int, body string) *http.Response {
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		StatusCode:    status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{},
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

// rateLimited builds a 429 response that asks for an immediate retry,
// so retry paths do not actually sleep.
func rateLimited() *http.Response {
	response := httpResponse(429, "")
	response.Header.Set("Retry-After", "0")
	return response
}

// serverError builds a 5xx response that asks for an immediate retry.
func serverError(status int) *http.Response {
	response := httpResponse(status, "")
	response.Header.Set("Retry-After", "0")
	return response
}

// errReader fails every read, to exercise body reading failures.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }
func (r errReader) Close() error             { return nil }
