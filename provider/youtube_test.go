package provider

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"bou.ke/monkey"
	"github.com/PuerkitoBio/goquery"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

var (
	result = youTubeResult{
		id:     "123",
		title:  "title",
		owner:  "owner",
		views:  1000000,
		length: 180,
	}
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

func TestYouTubeSearch(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
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
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")

	// testing
	assert.Nil(t, util.ErrOnly(youTube{}.Search(track)))
}

func TestYouTubeSearchMalformedData(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(strings.NewReader(
					fmt.Sprintf(resultScript, "", "", "", "", ""),
				)),
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")

	// testing
	assert.Nil(t, util.ErrOnly(youTube{}.Search(track)))
}

func TestYouTubeSearchNoData(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("<script>some unmatching script</script>")),
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")

	// testing
	assert.Nil(t, util.ErrOnly(youTube{}.Search(track)))
}

func TestYouTubeSearchFailingRequest(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")

	// testing
	assert.Error(t, util.ErrOnly(youTube{}.Search(track)), "failure")
}

func TestYouTubeSearchFailingRequestStatus(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")

	// testing
	assert.Error(t, util.ErrOnly(youTube{}.Search(track)))
}

func TestYouTubeSearchFailingGoQuery(t *testing.T) {
	// monkey patching
	monkey.Patch(goquery.NewDocumentFromReader, func(io.Reader) (*goquery.Document, error) {
		return nil, errors.New("failure")
	})
	defer monkey.Unpatch(goquery.NewDocumentFromReader)
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("<script>some unmatching script</script>")),
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")

	// testing
	assert.Error(t, util.ErrOnly(youTube{}.Search(track)), "failure")
}
