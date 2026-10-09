package provider

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

const (
	resultViewsText     = "1.000.000 views"
	resultLengthText    = "3:00 minutes"
	resultPublishedText = "1 year ago"
	resultScript        = `<script>var ytInitialData = {
		"contents": {
			"twoColumnSearchResultsRenderer": {
				"primaryContents": {
					"sectionListRenderer": {
						"contents": [{
							"itemSectionRenderer": {
								"contents": [{
									"videoRenderer": {
										"videoId": "%s",
										"title": {
											"runs": [{
												"text": "%s"
											}]
										},
										"ownerText": {
											"runs": [{
												"text": "%s"
											}]
										},
										"detailedMetadataSnippets": [{
											"snippetText": {
												"runs": [{
													"text": "%s"
												}]
											}
										}],
										"viewCountText": {
											"simpleText": "%s"
										},
										"lengthText": {
											"simpleText": "%s"
										},
										"publishedTimeText": {
											"simpleText": "%s"
										}
									}
								}]
							}
						}]
					}
				}
			}
		}
	}</script>`
)

var result = youTubeResult{
	id:          "123",
	title:       "title",
	owner:       "artist",
	description: misleading[0],
	views:       1000000,
	length:      180,
}

// youtubeResultPage renders the canned search results page carrying the
// shared result fixture.
func youtubeResultPage() string {
	return fmt.Sprintf(
		resultScript,
		result.id,
		result.title,
		result.owner,
		result.description,
		resultViewsText,
		resultLengthText,
		resultPublishedText,
	)
}

func BenchmarkYouTube(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestYouTubeSearch(&testing.T{})
	}
}

func TestYouTubeSearch(t *testing.T) {
	stubHTTP(t, func(*http.Request) (*http.Response, error) {
		return httpResponse(200, youtubeResultPage()), nil
	})

	// testing
	assert.Nil(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchMalformedData(t *testing.T) {
	stubHTTP(t, func(*http.Request) (*http.Response, error) {
		return httpResponse(200, `<script>var ytInitialData = {"content": {}`), nil
	})

	// testing
	assert.NotNil(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchPartialData(t *testing.T) {
	stubHTTP(t, func(*http.Request) (*http.Response, error) {
		return httpResponse(200, fmt.Sprintf(
			resultScript,
			result.id,
			"",
			"",
			"",
			"",
			"",
			"",
		)), nil
	})

	// testing
	assert.Nil(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchMaxRetriesExceeded(t *testing.T) {
	stubHTTP(t, func(*http.Request) (*http.Response, error) {
		response := httpResponse(429, "")
		response.Header.Set("Retry-After", "0")
		return response, nil
	})

	// testing
	assert.EqualError(t, sys.ErrOnly(youTube{}.search(track)), "youtube: max retries exceeded")
}

func TestYouTubeSearchTooManyRequests(t *testing.T) {
	callCount := 0
	stubHTTP(t, func(*http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			response := httpResponse(429, "")
			response.Header.Set("Retry-After", "0")
			return response, nil
		}
		return httpResponse(200, ""), nil
	})

	// testing
	assert.Nil(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchRedirectLoop(t *testing.T) {
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		response := httpResponse(302, "")
		response.Header.Set("Location", request.URL.String())
		return response, nil
	})

	// testing: captcha redirect fails fast without retrying
	assert.EqualError(t, sys.ErrOnly(youTube{}.search(track)), "youtube: blocked by google captcha")
}

func TestYouTubeSearchNoData(t *testing.T) {
	stubHTTP(t, func(*http.Request) (*http.Response, error) {
		return httpResponse(200, "<script>some unmatching script</script>"), nil
	})

	// testing
	assert.Nil(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchFailingRequest(t *testing.T) {
	stubHTTP(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("ko")
	})

	// testing: the client wraps transport errors in a url.Error
	assert.ErrorContains(t, sys.ErrOnly(youTube{}.search(track)), "ko")
}

func TestYouTubeSearchFailingRequestStatus(t *testing.T) {
	stubHTTP(t, func(*http.Request) (*http.Response, error) {
		return httpResponse(500, ""), nil
	})

	// testing
	assert.Error(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchFailingGoQuery(t *testing.T) {
	stubHTTP(t, func(*http.Request) (*http.Response, error) {
		response := httpResponse(200, "")
		response.Body = errReader{errors.New("ko")}
		return response, nil
	})

	// testing
	assert.EqualError(t, sys.ErrOnly(youTube{}.search(track)), "ko")
}

func TestScraping(t *testing.T) {
	if os.Getenv("TEST_SCRAPING") == "" {
		t.Skip("TEST_SCRAPING unset")
	}

	// the only test actually hitting the network: the keep-alive conn it
	// leaves in the default pool keeps an http2 readLoop goroutine around,
	// which goleak would report as a leak on package exit
	defer http.DefaultClient.CloseIdleConnections()

	// testing
	matches, err := youTube{}.search(&entity.Track{
		Title:    "White Christmas",
		Artists:  []string{"Bing Crosby"},
		Duration: 183,
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, matches)
}
