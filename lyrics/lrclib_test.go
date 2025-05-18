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
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

func BenchmarkLrclib(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestLrclibSearch(&testing.T{})
	}
}

func TestLrclibSearch(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(`{"syncedLyrics": "[00:27.37] lyrics", "plainLyrics": "lyrics"}`)),
		}, nil
	}).Reset()

	// testing
	lyrics, err := lrclib{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("[00:27.37]lyrics"), lyrics)
}

func TestLrclibSearchPlain(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(`{"plainLyrics": "lyrics"}`)),
		}, nil
	}).Reset()

	// testing
	lyrics, err := lrclib{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("lyrics"), lyrics)
}

func TestLrclibSearchNewRequestFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(http.NewRequestWithContext, func() (*http.Request, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(lrclib{}.search(track)), "ko")
}

func TestLrclibSearchNewRequestContextCanceled(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return nil, context.Canceled
	}).Reset()

	// testing
	lyrics, err := lrclib{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Nil(t, lyrics)
}

func TestLrclibSearchFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(lrclib{}.search(track)), "ko")
}

func TestLrclibSearchNotFound(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
			Body: io.NopCloser(
				strings.NewReader("")),
		}, nil
	}).Reset()

	// testing
	lyrics, err := lrclib{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestLrclibSearchTooManyRequests(t *testing.T) {
	// monkey patching
	doCounter := 0
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
			doCounter++
			if doCounter > 1 {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(
						strings.NewReader(`{"syncedLyrics": "[00:27.37] lyrics", "plainLyrics": "lyrics"}`)),
				}, nil
			}
			return &http.Response{
				StatusCode: 429,
				Body: io.NopCloser(
					strings.NewReader("")),
			}, nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(lrclib{}.search(track)))
}

func TestLrclibSearchInternalError(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 500,
			Body: io.NopCloser(
				strings.NewReader("")),
		}, nil
	}).Reset()

	// testing
	assert.NotNil(t, util.ErrOnly(lrclib{}.search(track)))
}

func TestLrclibSearchReadFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"syncedLyrics": "[00:27.37] lyrics", "plainLyrics": "lyrics"}`)),
			}, nil
		}).
		ApplyFunc(io.ReadAll, func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(lrclib{}.search(track)), "ko")
}

func TestLrclibSearchJsonFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"syncedLyrics": "[00:27.37] lyrics", "plainLyrics": "lyrics"}`)),
			}, nil
		}).
		ApplyFunc(json.Unmarshal, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(lrclib{}.search(track)), "ko")
}
