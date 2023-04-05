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
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
)

func TestDownload(t *testing.T) {
	// monkey patching
	monkey.Patch(os.ReadFile, func(string) ([]byte, error) { return nil, errors.New("not exists") })
	defer monkey.Unpatch(os.ReadFile)
	monkey.Patch(os.MkdirAll, func(string, fs.FileMode) error { return nil })
	defer monkey.Unpatch(os.MkdirAll)
	monkey.Patch(cmd.YouTubeDl, func(url, path string) error { return nil })
	defer monkey.Unpatch(cmd.YouTubeDl)
	monkey.PatchInstanceMethod(reflect.TypeOf(youTubeDl{}), "Download",
		func(youTubeDl, string, string, ...chan []byte) error { return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(youTubeDl{}), "Download")
	monkey.PatchInstanceMethod(reflect.TypeOf(youTubeDl{}), "Supports",
		func(youTubeDl, string) bool { return true })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(youTubeDl{}), "Supports")
	monkey.PatchInstanceMethod(reflect.TypeOf(blob{}), "Download",
		func(blob, string, string, ...chan []byte) error { return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(blob{}), "Download")
	monkey.PatchInstanceMethod(reflect.TypeOf(blob{}), "Supports",
		func(blob, string) bool { return false })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(blob{}), "Supports")

	// testing
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, Download("http://youtu.be", "fname.txt", ch))
}

func TestDownloadAlreadyExists(t *testing.T) {
	// monkey patching
	monkey.Patch(os.ReadFile, func(string) ([]byte, error) { return []byte{}, nil })
	defer monkey.Unpatch(os.ReadFile)

	// testing
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, Download("http://youtu.be", "fname.txt", ch))
}

func TestDownloadMakeDirFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(os.ReadFile, func(string) ([]byte, error) { return nil, errors.New("not exists") })
	defer monkey.Unpatch(os.ReadFile)
	monkey.Patch(os.MkdirAll, func(string, fs.FileMode) error { return errors.New("failure") })
	defer monkey.Unpatch(os.MkdirAll)

	// testing
	assert.Error(t, Download("http://youtu.be", "fname.txt"), "failure")
}

func TestDownloadUnsupported(t *testing.T) {
	// monkey patching
	monkey.Patch(os.ReadFile, func(string) ([]byte, error) { return nil, errors.New("not exists") })
	defer monkey.Unpatch(os.ReadFile)
	monkey.Patch(cmd.YouTubeDl, func(url, path string) error { return nil })
	defer monkey.Unpatch(cmd.YouTubeDl)
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
	assert.Error(t, Download("http://davidepucci.it", "fname.txt"), "unsupported url")
}
