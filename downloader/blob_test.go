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

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestBlobSupports(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientHead := gomonkey.ApplyMethod(reflect.TypeOf(http.DefaultClient), "Head",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		})
	defer patchhttpDefaultClientHead.Reset()

	// testing
	assert.True(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobSupportsError(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientHead := gomonkey.ApplyMethod(reflect.TypeOf(http.DefaultClient), "Head",
		func(*http.Client, string) (*http.Response, error) {
			return nil, errors.New("failure")
		})
	defer patchhttpDefaultClientHead.Reset()

	// testing
	assert.False(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobSupportsNotFound(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientHead := gomonkey.ApplyMethod(reflect.TypeOf(http.DefaultClient), "Head",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	defer patchhttpDefaultClientHead.Reset()

	// testing
	assert.False(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobUnsupported(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientHead := gomonkey.ApplyMethod(reflect.TypeOf(http.DefaultClient), "Head",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     map[string][]string{"Content-Type": {"text/plain"}},
			}, nil
		})
	defer patchhttpDefaultClientHead.Reset()

	// testing
	assert.False(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobDownload(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientGet := gomonkey.ApplyMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("bitch")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		})
	defer patchhttpDefaultClientGet.Reset()
	patchosOpenFile := gomonkey.ApplyFunc(os.OpenFile, func(string, int, fs.FileMode) (*os.File, error) {
		return nil, nil
	})
	defer patchosOpenFile.Reset()
	patchioReadAll := gomonkey.ApplyFunc(io.ReadAll, func(r io.Reader) ([]byte, error) {
		return []byte{}, nil
	})
	defer patchioReadAll.Reset()
	patchosFileWrite := gomonkey.ApplyMethod(reflect.TypeOf(&os.File{}), "Write",
		func(*os.File, []byte) (int, error) {
			return 0, nil
		})
	defer patchosFileWrite.Reset()

	// testing
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, blob{}.download("http://davidepucci.it", "/dev/null", ch))
}

func TestBlobDownloadFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientGet := gomonkey.ApplyMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(*http.Client, string) (*http.Response, error) {
			return nil, errors.New("failure")
		})
	defer patchhttpDefaultClientGet.Reset()

	// testing
	assert.Error(t, blob{}.download("http://davidepucci.it", "/dev/null"), "failure")
}

func TestBlobDownloadNotFound(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientGet := gomonkey.ApplyMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	defer patchhttpDefaultClientGet.Reset()

	// testing
	assert.NotNil(t, blob{}.download("http://davidepucci.it", "/dev/null"))
}

func TestBlobDownloadFileCreationFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientGet := gomonkey.ApplyMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		})
	defer patchhttpDefaultClientGet.Reset()
	patchosOpenFile := gomonkey.ApplyFunc(os.OpenFile, func(string, int, fs.FileMode) (*os.File, error) {
		return nil, errors.New("failure")
	})
	defer patchosOpenFile.Reset()

	// testing
	assert.Error(t, blob{}.download("http://davidepucci.it", "/dev/null"), "failure")
}

func TestBlobDownloadReadFailure(t *testing.T) {
	// monkey patching
	patchhttpDefaultClientGet := gomonkey.ApplyMethod(reflect.TypeOf(http.DefaultClient), "Get",
		func(*http.Client, string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		})
	defer patchhttpDefaultClientGet.Reset()
	patchosOpenFile := gomonkey.ApplyFunc(os.OpenFile, func(string, int, fs.FileMode) (*os.File, error) {
		return nil, nil
	})
	defer patchosOpenFile.Reset()
	patchioReadAll := gomonkey.ApplyFunc(io.ReadAll, func(r io.Reader) ([]byte, error) {
		return nil, errors.New("failure")
	})
	defer patchioReadAll.Reset()

	// testing
	assert.Error(t, blob{}.download("http://davidepucci.it", "/dev/null"), "failure")
}
