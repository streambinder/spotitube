package lyrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/agiledragon/gomonkey/v2"
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
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(client *http.Client, request *http.Request) (*http.Response, error) {
			if strings.EqualFold(request.Host, "api.genius.com") {
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
	defer patchhttpDefaultClientDo.Reset()

	// testing
	lyrics, err := genius{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("verse\nlyrics"), lyrics)
}

func TestGeniusSearchNewRequestFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultNewRequestWithContext := gomonkey.ApplyFunc(http.NewRequestWithContext,
		func(context.Context, string, string, io.Reader) (*http.Request, error) {
			return nil, errors.New("failure")
		})
	defer patchhttpDefaultNewRequestWithContext.Reset()

	// testing
	assert.Error(t, util.ErrOnly(genius{}.search(track)), "failure")
}

func TestGeniusSearchFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(*http.Client, *http.Request) (*http.Response, error) {
			return nil, errors.New("failure")
		})
	defer patchhttpDefaultClientDo.Reset()

	// testing
	assert.Error(t, util.ErrOnly(genius{}.search(track)), "failure")
}

func TestGeniusSearchHttpNotFound(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(*http.Client, *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body: io.NopCloser(
					strings.NewReader("")),
			}, nil
		})
	defer patchhttpDefaultClientDo.Reset()

	// testing
	assert.NotNil(t, util.ErrOnly(genius{}.search(track)))
}

func TestGeniusSearchReadFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(*http.Client, *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
			}, nil
		})
	defer patchhttpDefaultClientDo.Reset()
	patchioReadAll := gomonkey.ApplyFunc(io.ReadAll, func(r io.Reader) ([]byte, error) {
		return nil, errors.New("failure")
	})
	defer patchioReadAll.Reset()

	// testing
	assert.Error(t, util.ErrOnly(genius{}.search(track)), "failure")
}

func TestGeniusSearchNotFound(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(*http.Client, *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"response": {"hits": []}`)),
			}, nil
		})
	defer patchhttpDefaultClientDo.Reset()

	// testing
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestGeniusLyricsGetFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(client *http.Client, request *http.Request) (*http.Response, error) {
			if strings.EqualFold(request.Host, "api.genius.com") {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(
						strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
				}, nil
			}
			return nil, errors.New("failure")
		})
	defer patchhttpDefaultClientDo.Reset()

	// testing
	assert.Error(t, util.ErrOnly(genius{}.search(track)), "failure")
}

func TestGeniusLyricsNewRequestFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultNewRequestWithContext := gomonkey.ApplyFunc(http.NewRequestWithContext,
		func(context.Context, string, string, io.Reader) (*http.Request, error) {
			return nil, errors.New("failure")
		})
	defer patchhttpDefaultNewRequestWithContext.Reset()

	// testing
	assert.Error(t, util.ErrOnly(genius{}.fromGeniusURL("http://genius.com/test", context.Background())), "failure")
}

func TestGeniusLyricsNotFound(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(client *http.Client, request *http.Request) (*http.Response, error) {
			if strings.EqualFold(request.Host, "api.genius.com") {
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
	defer patchhttpDefaultClientDo.Reset()

	// testing
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.NotNil(t, err)
}

func TestGeniusLyricsNotParseable(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(client *http.Client, request *http.Request) (*http.Response, error) {
			if strings.EqualFold(request.Host, "api.genius.com") {
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
	defer patchhttpDefaultClientDo.Reset()
	patchgoqueryNewDocumentFromReader := gomonkey.ApplyFunc(goquery.NewDocumentFromReader,
		func(r io.Reader) (*goquery.Document, error) {
			return nil, errors.New("failure")
		})
	defer patchgoqueryNewDocumentFromReader.Reset()

	// testing
	assert.Error(t, util.ErrOnly(genius{}.search(track)), "failure")
}

func TestScraping(t *testing.T) {
	if os.Getenv("TEST_SCRAPING") == "" {
		return
	}

	// testing
	lyrics, err := genius{}.search(&entity.Track{
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
