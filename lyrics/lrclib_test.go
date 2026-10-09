package lyrics

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

const lrclibSyncedResponse = `{"syncedLyrics": "[00:27.37] lyrics", "plainLyrics": "lyrics"}`

func BenchmarkLrclib(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestLrclibSearch(&testing.T{})
	}
}

func TestLrclibSearch(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return httpResponse(200, lrclibSyncedResponse), nil
	})

	// testing: synced lyrics win and their whitespace is normalized
	lyrics, err := lrclib{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("[00:27.37]lyrics"), lyrics)
}

func TestLrclibSearchPlain(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return httpResponse(200, `{"plainLyrics": "lyrics"}`), nil
	})

	// testing
	lyrics, err := lrclib{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("lyrics"), lyrics)
}

func TestLrclibSearchContextCanceled(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return nil, context.Canceled
	})

	// testing: a cancelled context is a silent no-result
	lyrics, err := lrclib{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestLrclibSearchFailure(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("ko")
	})

	// testing: the client wraps transport errors in a url.Error
	assert.ErrorContains(t, sys.ErrOnly(lrclib{}.search(track)), "ko")
}

func TestLrclibSearchNotFound(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return httpResponse(404, ""), nil
	})

	// testing: a 404 is a silent no-result
	lyrics, err := lrclib{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestLrclibSearchMaxRetriesExceeded(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return rateLimited(), nil
	})

	// testing
	assert.EqualError(t, sys.ErrOnly(lrclib{}.search(track)), "lrclib: max retries exceeded")
}

func TestLrclibSearchTooManyRequests(t *testing.T) {
	callCount := 0
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return rateLimited(), nil
		}
		return httpResponse(200, lrclibSyncedResponse), nil
	})

	// testing: the search recovers after a 429
	lyrics, err := lrclib{}.search(track)
	assert.Nil(t, err)
	assert.Equal(t, []byte("[00:27.37]lyrics"), lyrics)
}

func TestLrclibSearchInternalError(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return serverError(500), nil
	})

	// testing
	assert.NotNil(t, sys.ErrOnly(lrclib{}.search(track)))
}

func TestLrclibSearchReadFailure(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		response := httpResponse(200, "")
		response.Body = errReader{errors.New("ko")}
		return response, nil
	})

	// testing
	assert.EqualError(t, sys.ErrOnly(lrclib{}.search(track)), "ko")
}

func TestLrclibSearchJsonFailure(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return httpResponse(200, "not json"), nil
	})

	// testing
	assert.NotNil(t, sys.ErrOnly(lrclib{}.search(track)))
}

func TestLrclibSearchServerErrorRetryOnce(t *testing.T) {
	callCount := 0
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		callCount++
		return serverError(503), nil
	})

	// testing: a server error is retried exactly once before failing
	assert.EqualError(t, sys.ErrOnly(lrclib{}.search(track)), "cannot fetch results on lrclib: 503 Service Unavailable")
	assert.Equal(t, 2, callCount)
}

func TestLrclibSearchServerErrorRecover(t *testing.T) {
	callCount := 0
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return serverError(503), nil
		}
		return httpResponse(200, lrclibSyncedResponse), nil
	})

	// testing: the single server-error retry can still succeed
	lyrics, err := lrclib{}.search(track)
	assert.Nil(t, err)
	assert.Equal(t, []byte("[00:27.37]lyrics"), lyrics)
	assert.Equal(t, 2, callCount)
}

func TestLrclibSearchBadRequest(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return httpResponse(400, ""), nil
	})

	// testing: a non-retryable status fails immediately
	assert.EqualError(t, sys.ErrOnly(lrclib{}.search(track)), "cannot fetch results on lrclib: 400 Bad Request")
}
