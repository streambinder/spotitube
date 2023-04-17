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
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
)

func TestDownload(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) { return nil, errors.New("not exists") })
	defer patchosReadFile.Reset()
	patchosMkdirAll := gomonkey.ApplyFunc(os.MkdirAll, func(string, fs.FileMode) error { return nil })
	defer patchosMkdirAll.Reset()
	patchcmdYouTubeDl := gomonkey.ApplyFunc(cmd.YouTubeDl, func(url, path string) error { return nil })
	defer patchcmdYouTubeDl.Reset()
	patchyouTubeDlDownload := gomonkey.ApplyPrivateMethod(reflect.TypeOf(youTubeDl{}), "download",
		func(youTubeDl, string, string, ...chan []byte) error { return nil })
	defer patchyouTubeDlDownload.Reset()
	patchyouTubeDlSupports := gomonkey.ApplyPrivateMethod(reflect.TypeOf(youTubeDl{}), "supports",
		func(youTubeDl, string) bool { return true })
	defer patchyouTubeDlSupports.Reset()
	patchblobDownload := gomonkey.ApplyPrivateMethod(reflect.TypeOf(blob{}), "download",
		func(blob, string, string, ...chan []byte) error { return nil })
	defer patchblobDownload.Reset()
	patchblobSupports := gomonkey.ApplyPrivateMethod(reflect.TypeOf(blob{}), "supports",
		func(blob, string) bool { return false })
	defer patchblobSupports.Reset()

	// testing
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, Download("http://youtu.be", "fname.txt", ch))
}

func TestDownloadAlreadyExists(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) { return []byte{}, nil })
	defer patchosReadFile.Reset()

	// testing
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, Download("http://youtu.be", "fname.txt", ch))
}

func TestDownloadMakeDirFailure(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) { return nil, errors.New("not exists") })
	defer patchosReadFile.Reset()
	patchosMkdirAll := gomonkey.ApplyFunc(os.MkdirAll, func(string, fs.FileMode) error { return errors.New("failure") })
	defer patchosMkdirAll.Reset()

	// testing
	assert.Error(t, Download("http://youtu.be", "fname.txt"), "failure")
}

func TestDownloadUnsupported(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) { return nil, errors.New("not exists") })
	defer patchosReadFile.Reset()
	patchcmdYouTubeDl := gomonkey.ApplyFunc(cmd.YouTubeDl, func(url, path string) error { return nil })
	defer patchcmdYouTubeDl.Reset()
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
	assert.Error(t, Download("http://davidepucci.it", "fname.txt"), "unsupported url")
}
