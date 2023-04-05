package downloader

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
)

func TestBlobSupports(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Head",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do")

	// testing
	assert.True(t, blob{}.Supports("http://davidepucci.it"))
}

func TestBlobSupportsError(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Head",
		func(*http.Client, string) (*http.Response, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do")

	// testing
	assert.False(t, blob{}.Supports("http://davidepucci.it"))
}

func TestBlobSupportsNotFound(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Head",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do")

	// testing
	assert.False(t, blob{}.Supports("http://davidepucci.it"))
}

func TestBlobUnsupported(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Head",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     map[string][]string{"Content-Type": {"text/plain"}},
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do")

	// testing
	assert.False(t, blob{}.Supports("http://davidepucci.it"))
}

func TestBlobDownload(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do")
	monkey.Patch(os.OpenFile, func(string, int, fs.FileMode) (*os.File, error) {
		return nil, nil
	})
	defer monkey.Unpatch(os.OpenFile)
	monkey.Patch(io.Copy, func(io.Writer, io.Reader) (int64, error) {
		return 0, nil
	})
	defer monkey.Unpatch(io.Copy)

	// testing
	assert.Nil(t, blob{}.Download("http://davidepucci.it", "/dev/null"))
}

func TestBlobDownloadFailure(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(*http.Client, string) (*http.Response, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do")

	// testing
	assert.Error(t, blob{}.Download("http://davidepucci.it", "/dev/null"), "failure")
}

func TestBlobDownloadNotFound(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do")

	// testing
	assert.NotNil(t, blob{}.Download("http://davidepucci.it", "/dev/null"))
}

func TestBlobDownloadFileCreationFailure(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do")
	monkey.Patch(os.OpenFile, func(string, int, fs.FileMode) (*os.File, error) {
		return nil, errors.New("failure")
	})
	defer monkey.Unpatch(os.OpenFile)

	// testing
	assert.Error(t, blob{}.Download("http://davidepucci.it", "/dev/null"), "failure")
}
