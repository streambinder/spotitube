package downloader

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/processor"
	"github.com/stretchr/testify/assert"
)

type mockProcessor struct {
	processor.Processor
	applies bool
	err     error
}

func (p mockProcessor) Applies(interface{}) bool {
	return p.applies
}

func (p mockProcessor) Do(interface{}) error {
	return p.err
}

func stubProcessor(applies bool, err error) processor.Processor {
	return mockProcessor{applies: applies, err: err}
}

func BenchmarkBlob(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestBlobSupports(&testing.T{})
		TestBlobDownload(&testing.T{})
	}
}

func TestBlobSupports(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(http.DefaultClient, "Head", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
		}, nil
	}).Reset()

	// testing
	assert.True(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobSupportsError(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(http.DefaultClient, "Head", func() (*http.Response, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.False(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobSupportsNotFound(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(http.DefaultClient, "Head", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}).Reset()

	// testing
	assert.False(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobUnsupported(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(http.DefaultClient, "Head", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     map[string][]string{"Content-Type": {"text/plain"}},
		}, nil
	}).Reset()

	// testing
	assert.False(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobDownload(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("bitch")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		}).
		ApplyFunc(os.OpenFile, func() (*os.File, error) {
			return nil, nil
		}).
		ApplyFunc(io.ReadAll, func() ([]byte, error) {
			return []byte{}, nil
		}).
		ApplyMethod(&os.File{}, "Write", func() (int, error) {
			return 0, nil
		}).
		Reset()

	// testing
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, blob{}.download("http://davidepucci.it", "/dev/null", stubProcessor(true, nil), ch))
}

func TestBlobDownloadProcessorFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("bitch")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		}).
		ApplyFunc(os.OpenFile, func() (*os.File, error) {
			return nil, nil
		}).
		ApplyFunc(io.ReadAll, func() ([]byte, error) {
			return []byte{}, nil
		}).
		Reset()

	// testing
	assert.Error(t, blob{}.download("http://davidepucci.it", "/dev/null", stubProcessor(true, errors.New("ko"))), "ko")
}

func TestBlobDownloadFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.Error(t, blob{}.download("http://davidepucci.it", "/dev/null", nil), "ko")
}

func TestBlobDownloadNotFound(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}).Reset()

	// testing
	assert.NotNil(t, blob{}.download("http://davidepucci.it", "/dev/null", nil))
}

func TestBlobDownloadFileCreationFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		}).
		ApplyFunc(os.OpenFile, func() (*os.File, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, blob{}.download("http://davidepucci.it", "/dev/null", nil), "ko")
}

func TestBlobDownloadReadFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyMethod(http.DefaultClient, "Get", func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
			}, nil
		}).
		ApplyFunc(os.OpenFile, func() (*os.File, error) {
			return nil, nil
		}).
		ApplyFunc(io.ReadAll, func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, blob{}.download("http://davidepucci.it", "/dev/null", nil), "ko")
}
