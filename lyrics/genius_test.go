package lyrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/agiledragon/gomonkey/v2"
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
)

func BenchmarkGenius(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestGeniusSearch(&testing.T{})
	}
}

func TestGeniusSearch(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func(_ *http.Client, request *http.Request) (*http.Response, error) {
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
	}).Reset()

	// testing
	lyrics, err := genius{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("verse\nlyrics"), lyrics)
}

func TestGeniusSearchNewRequestFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(http.NewRequestWithContext, func() (*http.Request, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusSearchNewRequestContextCanceled(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return nil, context.Canceled
	}).Reset()

	// testing
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestGeniusSearchMalformedData(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(`{"response": {}`)),
		}, nil
	}).Reset()

	// testing
	assert.Error(t, sys.ErrOnly(genius{}.search(track)))
}

func TestGeniusSearchFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusSearchHttpNotFound(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
			Body: io.NopCloser(
				strings.NewReader("")),
		}, nil
	}).Reset()

	// testing
	assert.NotNil(t, sys.ErrOnly(genius{}.search(track)))
}

func TestGeniusSearchTooManyRequests(t *testing.T) {
	// monkey patching
	var (
		doAPICounter            = 0
		doCounter               = 0
		tooManyRequestsResponse = &http.Response{
			StatusCode: 429,
			Body: io.NopCloser(
				strings.NewReader("")),
		}
	)
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func(_ *http.Client, request *http.Request) (*http.Response, error) {
			if strings.EqualFold(request.Host, "api.genius.com") {
				doAPICounter++
				if doAPICounter > 1 {
					return &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(
							strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
					}, nil
				}
				return tooManyRequestsResponse, nil
			}
			doCounter++
			if doCounter > 1 {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(
						strings.NewReader(`<div data-lyrics-container="true">verse<br/><span>lyrics</span></div>`)),
				}, nil
			}
			return tooManyRequestsResponse, nil
		}).
		Reset()

	// testing
	assert.Nil(t, sys.ErrOnly(genius{}.search(track)))
}

func TestGeniusSearchReadFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
			}, nil
		}).
		ApplyFunc(io.ReadAll, func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusSearchNotFound(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(`{"response": {"hits": []}}`)),
		}, nil
	}).Reset()

	// testing
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestGeniusLyricsGetFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func(_ *http.Client, request *http.Request) (*http.Response, error) {
		if strings.EqualFold(request.Host, "api.genius.com") {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
			}, nil
		}
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusLyricsNewRequestFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(http.NewRequestWithContext, func() (*http.Request, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.get("http://genius.com/test", context.Background())), "ko")
}

func TestGeniusLyricsNewRequestContextCanceled(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func(_ *http.Client, request *http.Request) (*http.Response, error) {
		if strings.EqualFold(request.Host, "api.genius.com") {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
			}, nil
		}
		return nil, context.Canceled
	}).Reset()

	// testing
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestGeniusLyricsNotFound(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func(_ *http.Client, request *http.Request) (*http.Response, error) {
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
	}).Reset()

	// testing
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.NotNil(t, err)
}

func TestGeniusLyricsNotParseable(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func(_ *http.Client, request *http.Request) (*http.Response, error) {
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
		}).
		ApplyFunc(goquery.NewDocumentFromReader, func() (*goquery.Document, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

// func TestScraping(t *testing.T) {
// 	if os.Getenv("TEST_SCRAPING") == "" {
// 		return
// 	}

// 	// testing
// 	lyrics, err := genius{}.search(&entity.Track{
// 		Title:   "White Christmas",
// 		Artists: []string{"Bing Crosby"},
// 	})
// 	assert.Nil(t, err)
// 	assert.Equal(t, []byte(`[Verse 1: Bing Crosby]
// I'm dreaming of a white Christmas
// Just like the ones I used to know
// Where the treetops glisten and children listen
// To hear sleigh bells in the snow

// [Verse 2: Bing Crosby]
// I'm dreaming of a white Christmas
// With every Christmas card I write
// "May your days be merry and bright
// And may all your Christmases be white"

// [Verse 3: Bing Crosby & Ken Darby Singers]
// I'm dreaming of a white Christmas
// Just like the ones I used to know
// Where the treetops glisten and children listen
// To hear sleigh bells in the snow

// [Verse 4: Bing Crosby & Ken Darby Singers, Bing Crosby]
// I'm dreaming of a white Christmas
// With every Christmas card I write
// "May your days be merry and bright
// And may all your Christmases be white"`), lyrics)
// }
