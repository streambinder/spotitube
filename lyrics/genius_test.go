package lyrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

const (
	response = `{
		"response": {
			"hits": [{
				"result": {
					"url": "https://genius.com/test",
					"title": "%s",
					"primary_artist": {"name": "%s"}
				}
			}]
		}
	}`
	lyricsPage = `<div data-lyrics-container="true">verse<br/><span>lyrics</span></div>`
)

// geniusRouter routes the search API and the lyrics pages to the given
// per-host handlers.
type geniusRouter struct {
	api  func(*http.Request) (*http.Response, error)
	page func(*http.Request) (*http.Response, error)
}

func (router geniusRouter) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host == "api.genius.com" {
		return router.api(request)
	}
	return router.page(request)
}

func geniusSearchOK(*http.Request) (*http.Response, error) {
	return httpResponse(200, fmt.Sprintf(response, track.Title, track.Artist())), nil
}

func geniusPageOK(*http.Request) (*http.Response, error) {
	return httpResponse(200, lyricsPage), nil
}

func BenchmarkGenius(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestGeniusSearch(&testing.T{})
	}
}

func TestGeniusSearch(t *testing.T) {
	stubRoundTripper(t, geniusRouter{geniusSearchOK, geniusPageOK})

	// testing
	lyrics, err := genius{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("verse\nlyrics"), lyrics)
}

func TestGeniusSearchContextCanceled(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return nil, context.Canceled
	})

	// testing: a cancelled context is a silent no-result
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestGeniusSearchMalformedData(t *testing.T) {
	stubRoundTripper(t, geniusRouter{func(*http.Request) (*http.Response, error) {
		return httpResponse(200, `{"response": {}`), nil
	}, geniusPageOK})

	// testing
	assert.Error(t, sys.ErrOnly(genius{}.search(track)))
}

func TestGeniusSearchFailure(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("ko")
	})

	// testing: the client wraps transport errors in a url.Error
	assert.ErrorContains(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusSearchHttpNotFound(t *testing.T) {
	stubRoundTripper(t, geniusRouter{func(*http.Request) (*http.Response, error) {
		return httpResponse(404, ""), nil
	}, geniusPageOK})

	// testing
	assert.NotNil(t, sys.ErrOnly(genius{}.search(track)))
}

func TestGeniusSearchMaxRetriesExceeded(t *testing.T) {
	stubRoundTripper(t, geniusRouter{func(*http.Request) (*http.Response, error) {
		return rateLimited(), nil
	}, geniusPageOK})

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "genius search: max retries exceeded")
}

func TestGeniusGetMaxRetriesExceeded(t *testing.T) {
	stubRoundTripper(t, geniusRouter{geniusSearchOK, func(*http.Request) (*http.Response, error) {
		return rateLimited(), nil
	}})

	// testing: the search succeeds, the lyrics page fetch keeps being
	// rate limited until its own retries are exhausted
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "genius get: max retries exceeded")
}

func TestGeniusSearchTooManyRequests(t *testing.T) {
	apiCount, pageCount := 0, 0
	stubRoundTripper(t, geniusRouter{
		func(request *http.Request) (*http.Response, error) {
			apiCount++
			if apiCount == 1 {
				return rateLimited(), nil
			}
			return geniusSearchOK(request)
		},
		func(request *http.Request) (*http.Response, error) {
			pageCount++
			if pageCount == 1 {
				return rateLimited(), nil
			}
			return geniusPageOK(request)
		},
	})

	// testing: both the search and the page fetch recover after a 429
	assert.Nil(t, sys.ErrOnly(genius{}.search(track)))
}

func TestGeniusSearchReadFailure(t *testing.T) {
	stubRoundTripper(t, geniusRouter{func(*http.Request) (*http.Response, error) {
		response := httpResponse(200, "")
		response.Body = errReader{errors.New("ko")}
		return response, nil
	}, geniusPageOK})

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusSearchNotFound(t *testing.T) {
	stubRoundTripper(t, geniusRouter{func(*http.Request) (*http.Response, error) {
		return httpResponse(200, `{"response":{"hits":[]}}`), nil
	}, geniusPageOK})

	// testing: no hits is a silent no-result, even after the
	// main-artist-only retry
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestGeniusLyricsGetFailure(t *testing.T) {
	stubRoundTripper(t, geniusRouter{geniusSearchOK, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("ko")
	}})

	// testing
	assert.ErrorContains(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusLyricsContextCanceled(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return nil, context.Canceled
	})

	// testing
	lyrics, err := genius{}.get("http://genius.com/test", context.Background())
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestGeniusLyricsNotFound(t *testing.T) {
	stubRoundTripper(t, geniusRouter{geniusSearchOK, func(*http.Request) (*http.Response, error) {
		return httpResponse(404, ""), nil
	}})

	// testing
	lyrics, err := genius{}.get("http://genius.com/test", context.Background())
	assert.Nil(t, lyrics)
	assert.NotNil(t, err)
}

func TestGeniusLyricsNotParseable(t *testing.T) {
	stubRoundTripper(t, geniusRouter{geniusSearchOK, func(*http.Request) (*http.Response, error) {
		response := httpResponse(200, "")
		response.Body = errReader{errors.New("ko")}
		return response, nil
	}})

	// testing: the lyrics page body cannot be read by the HTML parser
	assert.ErrorContains(t, sys.ErrOnly(genius{}.search(track)), "ko")
}
