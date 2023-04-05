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
	monkey.Patch(os.Stat, func(string) (fs.FileInfo, error) { return nil, errors.New("") })
	defer monkey.Unpatch(os.Stat)
	monkey.Patch(os.MkdirAll, func(string, fs.FileMode) error { return nil })
	defer monkey.Unpatch(os.MkdirAll)
	monkey.Patch(cmd.YouTubeDl, func(url, path string) error { return nil })
	defer monkey.Unpatch(cmd.YouTubeDl)
	monkey.PatchInstanceMethod(reflect.TypeOf(youTubeDl{}), "Download",
		func(youTubeDl, string, string) error { return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(youTubeDl{}), "Download")
	monkey.PatchInstanceMethod(reflect.TypeOf(youTubeDl{}), "Supports",
		func(youTubeDl, string) bool { return true })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(youTubeDl{}), "Supports")
	monkey.PatchInstanceMethod(reflect.TypeOf(blob{}), "Download",
		func(blob, string, string) error { return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(blob{}), "Download")
	monkey.PatchInstanceMethod(reflect.TypeOf(blob{}), "Supports",
		func(blob, string) bool { return false })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(blob{}), "Supports")

	// testing
	assert.Nil(t, Download("http://youtu.be", "fname.txt"))
}

func TestDownloadAlreadyExists(t *testing.T) {
	// monkey patching
	monkey.Patch(os.Stat, func(string) (fs.FileInfo, error) { return nil, nil })
	defer monkey.Unpatch(os.Stat)

	// testing
	assert.Nil(t, Download("http://youtu.be", "fname.txt"))
}

func TestDownloadMakeDirFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(os.Stat, func(string) (fs.FileInfo, error) { return nil, errors.New("") })
	defer monkey.Unpatch(os.Stat)
	monkey.Patch(os.MkdirAll, func(string, fs.FileMode) error { return errors.New("failure") })
	defer monkey.Unpatch(os.MkdirAll)

	// testing
	assert.Error(t, Download("http://youtu.be", "fname.txt"), "failure")
}

func TestDownloadUnsupported(t *testing.T) {
	// monkey patching
	monkey.Patch(os.Stat, func(string) (fs.FileInfo, error) { return nil, errors.New("") })
	defer monkey.Unpatch(os.Stat)
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
