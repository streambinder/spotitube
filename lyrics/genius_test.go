package lyrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/bytedance/mockey"
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
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, request *http.Request) (*http.Response, error) {
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
	}).Build()

	// testing
	lyrics, err := genius{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("verse\nlyrics"), lyrics)
}

func TestGeniusSearchNewRequestFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(http.NewRequestWithContext).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusSearchNewRequestContextCanceled(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).Return(nil, context.Canceled).Build()

	// testing
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestGeniusSearchMalformedData(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, _ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(`{"response": {}`)),
		}, nil
	}).Build()

	// testing
	assert.Error(t, sys.ErrOnly(genius{}.search(track)))
}

func TestGeniusSearchFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusSearchHttpNotFound(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, _ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
			Body: io.NopCloser(
				strings.NewReader("")),
		}, nil
	}).Build()

	// testing
	assert.NotNil(t, sys.ErrOnly(genius{}.search(track)))
}

func TestGeniusSearchMaxRetriesExceeded(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(sys.SleepUntilRetry).Return().Build()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, _ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 429,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "genius search: max retries exceeded")
}

func TestGeniusSearchRetryRequestBuildFailure(t *testing.T) {
	// monkey patching
	callCount := 0
	defer mockey.UnPatchAll()
	mockey.Mock(sys.SleepUntilRetry).Return().Build()
	mockey.Mock(http.NewRequestWithContext).To(func(_ context.Context, method, url string, _ io.Reader) (*http.Request, error) {
		callCount++
		if callCount > 1 {
			return nil, errors.New("ko")
		}
		req := &http.Request{Method: method, Header: make(http.Header)}
		parsedURL, parseErr := neturl.Parse(url)
		if parseErr != nil {
			return nil, parseErr
		}
		req.URL = parsedURL
		return req, nil
	}).Build()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, _ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 429,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusGetMaxRetriesExceeded(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(sys.SleepUntilRetry).Return().Build()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, request *http.Request) (*http.Response, error) {
		if strings.EqualFold(request.Host, "api.genius.com") {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
			}, nil
		}
		return &http.Response{
			StatusCode: 429,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "genius get: max retries exceeded")
}

func TestGeniusGetRetryRequestBuildFailure(t *testing.T) {
	// monkey patching
	callCount := 0
	defer mockey.UnPatchAll()
	mockey.Mock(sys.SleepUntilRetry).Return().Build()
	mockey.Mock(http.NewRequestWithContext).To(func(_ context.Context, method, url string, _ io.Reader) (*http.Request, error) {
		callCount++
		if callCount > 1 {
			return nil, errors.New("ko")
		}
		req := &http.Request{Method: method, Header: make(http.Header)}
		parsedURL, parseErr := neturl.Parse(url)
		if parseErr != nil {
			return nil, parseErr
		}
		req.URL = parsedURL
		return req, nil
	}).Build()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, _ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 429,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.get("http://localhost/")), "ko")
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
	defer mockey.UnPatchAll()
	mockey.Mock(sys.SleepUntilRetry).Return().Build()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, request *http.Request) (*http.Response, error) {
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
	}).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(genius{}.search(track)))
}

func TestGeniusSearchReadFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, _ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
		}, nil
	}).Build()
	mockey.Mock(io.ReadAll).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusSearchNotFound(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, _ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(`{"response": {"hits": []}}`)),
		}, nil
	}).Build()

	// testing
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestGeniusLyricsGetFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, request *http.Request) (*http.Response, error) {
		if strings.EqualFold(request.Host, "api.genius.com") {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
			}, nil
		}
		return nil, errors.New("ko")
	}).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.search(track)), "ko")
}

func TestGeniusLyricsNewRequestFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(http.NewRequestWithContext).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(genius{}.get("http://genius.com/test", context.Background())), "ko")
}

func TestGeniusLyricsNewRequestContextCanceled(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, request *http.Request) (*http.Response, error) {
		if strings.EqualFold(request.Host, "api.genius.com") {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(fmt.Sprintf(response, track.Title, track.Artists[0]))),
			}, nil
		}
		return nil, context.Canceled
	}).Build()

	// testing
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestGeniusLyricsNotFound(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, request *http.Request) (*http.Response, error) {
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
	}).Build()

	// testing
	lyrics, err := genius{}.search(track)
	assert.Nil(t, lyrics)
	assert.NotNil(t, err)
}

func TestGeniusLyricsNotParseable(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, request *http.Request) (*http.Response, error) {
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
	}).Build()
	mockey.Mock(goquery.NewDocumentFromReader).Return(nil, errors.New("ko")).Build()

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
