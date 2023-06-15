package downloader

import (
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
)

func BenchmarkDownload(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestDownload(&testing.T{})
	}
}

func TestDownload(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("not exists")
		}).
		ApplyFunc(os.MkdirAll, func() error {
			return nil
		}).
		ApplyFunc(cmd.YouTubeDl, func() error {
			return nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(youTubeDl{}), "download", func() error {
			return nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(youTubeDl{}), "supports", func() bool {
			return true
		}).
		ApplyPrivateMethod(reflect.TypeOf(blob{}), "download", func() error {
			return nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(blob{}), "supports", func() bool {
			return false
		}).
		Reset()

	// testing
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, Download("http://youtu.be", "fname.txt", nil, ch))
}

func TestDownloadEmpty(t *testing.T) {
	assert.Nil(t, Download("", "fname.txt", nil))
}

func TestDownloadAlreadyExists(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(os.ReadFile, func() ([]byte, error) {
		return []byte{}, nil
	}).Reset()

	// testing
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, Download("http://youtu.be", "fname.txt", nil, ch))
}

func TestDownloadMakeDirFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("not exists")
		}).
		ApplyFunc(os.MkdirAll, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, Download("http://youtu.be", "fname.txt", nil), "ko")
}

func TestDownloadUnsupported(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("not exists")
		}).
		ApplyFunc(cmd.YouTubeDl, func() error {
			return nil
		}).
		ApplyMethod(http.DefaultClient, "Head", func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     map[string][]string{"Content-Type": {"text/plain"}},
			}, nil
		}).
		Reset()

	// testing
	assert.Error(t, Download("http://davidepucci.it", "fname.txt", nil))
}
