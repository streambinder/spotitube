package lyrics

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"bou.ke/monkey"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

func TestLyricsOvhSearch(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")

	// testing
	lyrics, err := lyricsOvh{}.Search(track)
	assert.Nil(t, err)
	assert.Equal(t, []byte("lyrics"), lyrics)
}

func TestLyricsOvhSearchFailure(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.Search(track)), "failure")
}

func TestLyricsOvhSearchNotFound(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body: io.NopCloser(
					strings.NewReader("")),
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")

	// testing
	lyrics, err := lyricsOvh{}.Search(track)
	assert.Nil(t, lyrics)
	assert.Nil(t, err)
}

func TestLyricsOvhSearchInternalError(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 500,
				Body: io.NopCloser(
					strings.NewReader("")),
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")

	// testing
	assert.NotNil(t, util.ErrOnly(lyricsOvh{}.Search(track)))
}

func TestLyricsOvhSearchReadFailure(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")
	monkey.Patch(io.ReadAll, func(r io.Reader) ([]byte, error) {
		return nil, errors.New("failure")
	})
	defer monkey.Unpatch(io.ReadAll)

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.Search(track)), "failure")
}

func TestLyricsOvhSearchJsonFailure(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(client *http.Client, url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(
					strings.NewReader(`{"lyrics": "lyrics"}`)),
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get")
	monkey.Patch(json.Unmarshal, func(data []byte, v any) error {
		return errors.New("failure")
	})
	defer monkey.Unpatch(io.ReadAll)

	// testing
	assert.Error(t, util.ErrOnly(lyricsOvh{}.Search(track)), "failure")
}
