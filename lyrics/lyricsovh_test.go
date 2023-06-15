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

func BenchmarkLyricsOVH(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestLyricsOvhSearch(&testing.T{})
	}
}

func TestLyricsOvhSearch(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(`{"lyrics": "lyrics"}`)),
		}, nil
	}).Reset()

	// testing
	lyrics, err := lyricsOvh{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("lyrics"), lyrics)
}

func TestLyricsOvhSearchNewRequestFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(http.NewRequestWithContext, func() (*http.Request, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(lyricsOvh{}.search(track)), "ko")
}

func TestLyricsOvhSearchNewRequestContextCanceled(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return nil, context.Canceled
	}).Reset()

	// testing
	lyrics, err := lyricsOvh{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Nil(t, lyrics)
}

func TestLyricsOvhSearchFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(lyricsOvh{}.search(track)), "ko")
}

func TestLyricsOvhSearchNotFound(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
			Body: io.NopCloser(
				strings.NewReader("")),
		}, nil
	}).Reset()

	// testing
	lyrics, err := lyricsOvh{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestLyricsOvhSearchTooManyRequests(t *testing.T) {
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
						strings.NewReader(`{"lyrics": "lyrics"}`)),
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
	assert.Nil(t, util.ErrOnly(lyricsOvh{}.search(track)))
}

func TestLyricsOvhSearchInternalError(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 500,
			Body: io.NopCloser(
				strings.NewReader("")),
		}, nil
	}).Reset()

	// testing
	assert.NotNil(t, util.ErrOnly(lyricsOvh{}.search(track)))
}

func TestLyricsOvhSearchReadFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		}).
		ApplyFunc(io.ReadAll, func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(lyricsOvh{}.search(track)), "ko")
}

func TestLyricsOvhSearchJsonFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyPrivateMethod(reflect.TypeOf(http.DefaultClient), "do", func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		}).
		ApplyFunc(json.Unmarshal, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(lyricsOvh{}.search(track)), "ko")
}
