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
				Body:       io.NopCloser(strings.NewReader("bitch")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do")
	monkey.Patch(os.OpenFile, func(string, int, fs.FileMode) (*os.File, error) {
		return nil, nil
	})
	defer monkey.Unpatch(os.OpenFile)
	monkey.Patch(io.ReadAll, func(r io.Reader) ([]byte, error) {
		return []byte{}, nil
	})
	defer monkey.Unpatch(io.ReadAll)
	monkey.PatchInstanceMethod(reflect.TypeOf(&os.File{}), "Write",
		func(*os.File, []byte) (int, error) {
			return 0, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&os.File{}), "Write")

	// testing
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, blob{}.Download("http://davidepucci.it", "/dev/null", ch))
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

func TestBlobDownloadReadFailure(t *testing.T) {
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
	monkey.Patch(io.ReadAll, func(r io.Reader) ([]byte, error) {
		return nil, errors.New("failure")
	})
	defer monkey.Unpatch(io.ReadAll)

	// testing
	assert.Error(t, blob{}.Download("http://davidepucci.it", "/dev/null"), "failure")
}
