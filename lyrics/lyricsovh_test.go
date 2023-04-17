package lyrics

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

func TestLyricsOvhSearch(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(*http.Client, *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer patchhttpDefaultClientDo.Reset()

	// testing
	lyrics, err := lyricsOvh{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("lyrics"), lyrics)
}

func TestLyricsOvhSearchNewRequestFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultNewRequestWithContext := gomonkey.ApplyFunc(http.NewRequestWithContext,
		func(context.Context, string, string, io.Reader) (*http.Request, error) {
			return nil, errors.New("failure")
		})
	defer patchhttpDefaultNewRequestWithContext.Reset()

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.search(track)), "failure")
}

func TestLyricsOvhSearchFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(*http.Client, *http.Request) (*http.Response, error) {
			return nil, errors.New("failure")
		})
	defer patchhttpDefaultClientDo.Reset()

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.search(track)), "failure")
}

func TestLyricsOvhSearchNotFound(t *testing.T) {
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
	lyrics, err := lyricsOvh{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestLyricsOvhSearchInternalError(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(*http.Client, *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 500,
				Body: io.NopCloser(
					strings.NewReader("")),
			}, nil
		})
	defer patchhttpDefaultClientDo.Reset()

	// testing
	assert.NotNil(t, util.ErrOnly(lyricsOvh{}.search(track)))
}

func TestLyricsOvhSearchReadFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(*http.Client, *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer patchhttpDefaultClientDo.Reset()
	patchioReadAll := gomonkey.ApplyFunc(io.ReadAll, func(r io.Reader) ([]byte, error) {
		return nil, errors.New("failure")
	})
	defer patchioReadAll.Reset()

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.search(track)), "failure")
}

func TestLyricsOvhSearchJsonFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientDo := gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do",
		func(*http.Client, *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer patchhttpDefaultClientDo.Reset()
	patchjsonUnmarshal := gomonkey.ApplyFunc(json.Unmarshal, func(data []byte, v any) error {
		return errors.New("failure")
	})
	defer patchjsonUnmarshal.Reset()

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.search(track)), "failure")
}
