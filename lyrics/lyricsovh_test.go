package lyrics

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

func BenchmarkLyricsOVH(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestLyricsOvhSearch(&testing.T{})
	}
}

func TestLyricsOvhSearch(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, _ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(`{"lyrics": "lyrics"}`)),
		}, nil
	}).Build()

	// testing
	lyrics, err := lyricsOvh{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("lyrics"), lyrics)
}

func TestLyricsOvhSearchNewRequestFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(http.NewRequestWithContext).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(lyricsOvh{}.search(track)), "ko")
}

func TestLyricsOvhSearchNewRequestContextCanceled(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).Return(nil, context.Canceled).Build()

	// testing
	lyrics, err := lyricsOvh{}.search(track, context.Background())
	assert.Nil(t, err)
	assert.Nil(t, lyrics)
}

func TestLyricsOvhSearchFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(lyricsOvh{}.search(track)), "ko")
}

func TestLyricsOvhSearchNotFound(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).Return(&http.Response{
		StatusCode: 404,
		Body: io.NopCloser(
			strings.NewReader("")),
	}, nil).Build()

	// testing
	lyrics, err := lyricsOvh{}.search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestLyricsOvhSearchMaxRetriesExceeded(t *testing.T) {
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
	assert.EqualError(t, sys.ErrOnly(lyricsOvh{}.search(track)), "lyrics.ovh: max retries exceeded")
}

func TestLyricsOvhSearchTooManyRequests(t *testing.T) {
	// monkey patching
	doCounter := 0
	defer mockey.UnPatchAll()
	mockey.Mock(sys.SleepUntilRetry).Return().Build()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, _ *http.Request) (*http.Response, error) {
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
	}).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(lyricsOvh{}.search(track)))
}

func TestLyricsOvhSearchInternalError(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).Return(&http.Response{
		StatusCode: 500,
		Body: io.NopCloser(
			strings.NewReader("")),
	}, nil).Build()

	// testing
	assert.NotNil(t, sys.ErrOnly(lyricsOvh{}.search(track)))
}

func TestLyricsOvhSearchReadFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, _ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(`{"lyrics": "lyrics"}`)),
		}, nil
	}).Build()
	mockey.Mock(io.ReadAll).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(lyricsOvh{}.search(track)), "ko")
}

func TestLyricsOvhSearchJsonFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "do")).To(func(_ *http.Client, _ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(`{"lyrics": "lyrics"}`)),
		}, nil
	}).Build()
	mockey.Mock(json.Unmarshal).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(lyricsOvh{}.search(track)), "ko")
}
