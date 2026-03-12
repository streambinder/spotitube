package provider

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/bytedance/mockey"
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

func BenchmarkYouTube(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestYouTubeSearch(&testing.T{})
	}
}

func TestYouTubeSearch(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body: io.NopCloser(strings.NewReader(
			fmt.Sprintf(
				resultScript,
				result.id,
				result.title,
				result.owner,
				result.description,
				resultViewsText,
				resultLengthText,
				resultPublishedText,
			),
		)),
	}, nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchMalformedData(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`<script>var ytInitialData = {"content": {}`)),
	}, nil).Build()

	// testing
	assert.NotNil(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchPartialData(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body: io.NopCloser(strings.NewReader(fmt.Sprintf(
			resultScript,
			result.id,
			"",
			"",
			"",
			"",
			"",
			"",
		))),
	}, nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchMaxRetriesExceeded(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(sys.SleepUntilRetry).Return().Build()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).To(func(_ string) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader(""))}, nil
	}).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(youTube{}.search(track)), "youtube: max retries exceeded")
}

func TestYouTubeSearchTooManyRequests(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(sys.SleepUntilRetry).Return().Build()
	callCount := 0
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).To(func(_ string) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	}).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchRedirectLoop(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).To(func(_ string) (*http.Response, error) {
		return nil, errors.New(`Get "https://www.google.com/sorry/index?continue=...": stopped after 10 redirects`)
	}).Build()

	// testing: captcha redirect fails fast without retrying
	assert.EqualError(t, sys.ErrOnly(youTube{}.search(track)), "youtube: blocked by google captcha")
}

func TestYouTubeSearchNoData(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("<script>some unmatching script</script>")),
	}, nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchFailingRequest(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(youTube{}.search(track)), "ko")
}

func TestYouTubeSearchFailingRequestStatus(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(
		&http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}, nil,
	).Build()

	// testing
	assert.Error(t, sys.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchFailingGoQuery(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(goquery.NewDocumentFromReader).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("<script>some unmatching script</script>")),
	}, nil).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(youTube{}.search(track)), "ko")
}

func TestScraping(t *testing.T) {
	if os.Getenv("TEST_SCRAPING") == "" {
		return
	}

	// testing
	matches, err := youTube{}.search(&entity.Track{
		Title:    "White Christmas",
		Artists:  []string{"Bing Crosby"},
		Duration: 183,
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, matches)
}
