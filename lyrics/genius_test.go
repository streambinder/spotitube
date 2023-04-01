package lyrics

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"bou.ke/monkey"
	"github.com/PuerkitoBio/goquery"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

const (
	response = `{"response": {
		"hits": [{
			"result": {
				"url": "https://genius.com/test",
				"title": "%s",
				"primary_artist": {"name": "%s"}
			}
		}],
	}`
)

func TestGeniusSearch(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(method string, url string, queryParameters url.Values, body io.Reader, headers ...string) (*http.Response, error) {
			if strings.Contains(url, "//api.genius.com") {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(
						strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
				}, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`<div data-lyrics-container="true">verse<br/><span>lyrics</span></div>`)),
			}, nil
		})
	defer monkey.Unpatch(util.HttpRequest)

	// testing
	lyrics, err := genius{}.Search(track)
	assert.Nil(t, err)
	assert.Equal(t, []byte("verse\nlyrics"), lyrics)
}

func TestGeniusSearchFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return nil, errors.New("failure")
		})
	defer monkey.Unpatch(util.HttpRequest)

	// testing
	assert.Error(t, util.ErrOnly(genius{}.Search(track)), "failure")
}

func TestGeniusSearchReadFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(method string, url string, queryParameters url.Values, body io.Reader, headers ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
			}, nil
		})
	defer monkey.Unpatch(util.HttpRequest)
	monkey.Patch(io.ReadAll, func(r io.Reader) ([]byte, error) {
		return nil, errors.New("failure")
	})
	defer monkey.Unpatch(io.ReadAll)

	// testing
	assert.Error(t, util.ErrOnly(genius{}.Search(track)), "failure")
}

func TestGeniusSearchNotFound(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(method string, url string, queryParameters url.Values, body io.Reader, headers ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"response": {"hits": []}`)),
			}, nil
		})
	defer monkey.Unpatch(util.HttpRequest)

	// testing
	lyrics, err := genius{}.Search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestGeniusLyricsGetFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(method string, url string, queryParameters url.Values, body io.Reader, headers ...string) (*http.Response, error) {
			if strings.Contains(url, "//api.genius.com") {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(
						strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
				}, nil
			}
			return nil, errors.New("failure")
		})
	defer monkey.Unpatch(util.HttpRequest)

	// testing
	assert.Error(t, util.ErrOnly(genius{}.Search(track)), "failure")
}

func TestGeniusLyricsNotFound(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(method string, url string, queryParameters url.Values, body io.Reader, headers ...string) (*http.Response, error) {
			if strings.Contains(url, "//api.genius.com") {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(
						strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
				}, nil
			}
			return &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	defer monkey.Unpatch(util.HttpRequest)

	// testing
	lyrics, err := genius{}.Search(track)
	assert.Nil(t, lyrics)
	assert.NotNil(t, err)
}

func TestGeniusLyricsNotParseable(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(method string, url string, queryParameters url.Values, body io.Reader, headers ...string) (*http.Response, error) {
			if strings.Contains(url, "//api.genius.com") {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(
						strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
				}, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	defer monkey.Unpatch(util.HttpRequest)
	monkey.Patch(goquery.NewDocumentFromReader,
		func(r io.Reader) (*goquery.Document, error) {
			return nil, errors.New("failure")
		})
	defer monkey.Unpatch(util.HttpRequest)

	// testing
	assert.Error(t, util.ErrOnly(genius{}.Search(track)), "failure")
}

func TestScraping(t *testing.T) {
	if os.Getenv("TEST_SCRAPING") == "" {
		return
	}

	// testing
	lyrics, err := genius{}.Search(&entity.Track{
		Title:   "White Christmas",
		Artists: []string{"Bing Crosby"},
	})
	assert.Nil(t, err)
	assert.Equal(t, []byte(`[Verse 1: Bing Crosby]
I'm dreaming of a white Christmas
Just like the ones I used to know
Where the treetops glisten and children listen
To hear sleigh bells in the snow

[Verse 2: Bing Crosby]
I'm dreaming of a white Christmas
With every Christmas card I write
"May your days be merry and bright
And may all your Christmases be white"

[Verse 3: Bing Crosby & Ken Darby Singers]
I'm dreaming of a white Christmas
Just like the ones I used to know
Where the treetops glisten and children listen
To hear sleigh bells in the snow

[Verse 4: Bing Crosby & Ken Darby Singers, Bing Crosby]
I'm dreaming of a white Christmas
With every Christmas card I write
"May your days be merry and bright
And may all your Christmases be white"`), lyrics)
}
