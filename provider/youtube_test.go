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
	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

const (
	resultViewsText  = "1.000.000 views"
	resultLengthText = "3:00 minutes"
	resultScript     = `<script>var ytInitialData = {
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
										"viewCountText": {
											"simpleText": "%s"
										},
										"lengthText": {
											"simpleText": "%s"
										}
									}
								}]
							}]
						}
					}
				}
			}
		}
	</script>`
)

var result = youTubeResult{
	id:     "123",
	title:  "title",
	owner:  "artist",
	views:  1000000,
	length: 180,
}

func TestYouTubeSearch(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(
				fmt.Sprintf(
					resultScript,
					result.id,
					result.title,
					result.owner,
					resultViewsText,
					resultLengthText,
				),
			)),
		}, nil
	}).Reset()

	// testing
	assert.Nil(t, util.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchMalformedData(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(
				fmt.Sprintf(resultScript, "", "", "", "", ""),
			)),
		}, nil
	}).Reset()

	// testing
	assert.Nil(t, util.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchNoData(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("<script>some unmatching script</script>")),
		}, nil
	}).Reset()

	// testing
	assert.Nil(t, util.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchFailingRequest(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.Error(t, util.ErrOnly(youTube{}.search(track)), "ko")
}

func TestYouTubeSearchFailingRequestStatus(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}, nil
	}).Reset()

	// testing
	assert.Error(t, util.ErrOnly(youTube{}.search(track)))
}

func TestYouTubeSearchFailingGoQuery(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(goquery.NewDocumentFromReader, func() (*goquery.Document, error) {
			return nil, errors.New("ko")
		}).
		ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("<script>some unmatching script</script>")),
			}, nil
		}).
		Reset()

	// testing
	assert.Error(t, util.ErrOnly(youTube{}.search(track)), "ko")
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
