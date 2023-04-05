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

	"bou.ke/monkey"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

func TestLyricsOvhSearch(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer monkey.Unpatch(util.HttpRequest)

	// testing
	lyrics, err := lyricsOvh{}.Search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("lyrics"), lyrics)
}

func TestLyricsOvhSearchFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return nil, errors.New("failure")
		})
	defer monkey.Unpatch(util.HttpRequest)

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.Search(track)), "failure")
}

func TestLyricsOvhSearchNotFound(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body: io.NopCloser(
					strings.NewReader("")),
			}, nil
		})
	defer monkey.Unpatch(util.HttpRequest)

	// testing
	lyrics, err := lyricsOvh{}.Search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestLyricsOvhSearchInternalError(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 500,
				Body: io.NopCloser(
					strings.NewReader("")),
			}, nil
		})
	defer monkey.Unpatch(util.HttpRequest)

	// testing
	assert.NotNil(t, util.ErrOnly(lyricsOvh{}.Search(track)))
}

func TestLyricsOvhSearchReadFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer monkey.Unpatch(util.HttpRequest)
	monkey.Patch(io.ReadAll, func(r io.Reader) ([]byte, error) {
		return nil, errors.New("failure")
	})
	defer monkey.Unpatch(io.ReadAll)

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.Search(track)), "failure")
}

func TestLyricsOvhSearchJsonFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(util.HttpRequest,
		func(context.Context, string, string, url.Values, io.Reader, ...string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer monkey.Unpatch(util.HttpRequest)
	monkey.Patch(json.Unmarshal, func(data []byte, v any) error {
		return errors.New("failure")
	})
	defer monkey.Unpatch(io.ReadAll)

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.Search(track)), "failure")
}
