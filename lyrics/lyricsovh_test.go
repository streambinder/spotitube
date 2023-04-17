package lyrics

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

func TestLyricsOvhSearch(t *testing.T) {
	// monkey patching
	patchutilHttpRequest := gomonkey.ApplyFunc(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer patchutilHttpRequest.Reset()

	// testing
	lyrics, err := lyricsOvh{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("lyrics"), lyrics)
}

func TestLyricsOvhSearchFailure(t *testing.T) {
	// monkey patching
	patchutilHttpRequest := gomonkey.ApplyFunc(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return nil, errors.New("failure")
		})
	defer patchutilHttpRequest.Reset()

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.search(track)), "failure")
}

func TestLyricsOvhSearchNotFound(t *testing.T) {
	// monkey patching
	patchutilHttpRequest := gomonkey.ApplyFunc(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body: io.NopCloser(
					strings.NewReader("")),
			}, nil
		})
	defer patchutilHttpRequest.Reset()

	// testing
	lyrics, err := lyricsOvh{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestLyricsOvhSearchInternalError(t *testing.T) {
	// monkey patching
	patchutilHttpRequest := gomonkey.ApplyFunc(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 500,
				Body: io.NopCloser(
					strings.NewReader("")),
			}, nil
		})
	defer patchutilHttpRequest.Reset()

	// testing
	assert.NotNil(t, util.ErrOnly(lyricsOvh{}.search(track)))
}

func TestLyricsOvhSearchReadFailure(t *testing.T) {
	// monkey patching
	patchutilHttpRequest := gomonkey.ApplyFunc(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer patchutilHttpRequest.Reset()
	patchioReadAll := gomonkey.ApplyFunc(io.ReadAll, func(r io.Reader) ([]byte, error) {
		return nil, errors.New("failure")
	})
	defer patchioReadAll.Reset()

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.search(track)), "failure")
}

func TestLyricsOvhSearchJsonFailure(t *testing.T) {
	// monkey patching
	patchutilHttpRequest := gomonkey.ApplyFunc(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer patchutilHttpRequest.Reset()
	patchjsonUnmarshal := gomonkey.ApplyFunc(json.Unmarshal, func(data []byte, v any) error {
		return errors.New("failure")
	})
	defer patchjsonUnmarshal.Reset()

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.search(track)), "failure")
}
