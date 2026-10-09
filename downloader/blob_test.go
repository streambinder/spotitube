package downloader

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

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

// roundTripFunc adapts a function to http.RoundTripper, so tests can
// script the responses the blob downloader receives without any network.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

// stubTransport points the package http client at the given round tripper
// for the duration of the test.
func stubTransport(t *testing.T, trip func(*http.Request) (*http.Response, error)) {
	t.Helper()
	previous := httpClient.Transport
	httpClient.Transport = roundTripFunc(trip)
	t.Cleanup(func() { httpClient.Transport = previous })
}

func blobResponse(status int, contentType, body string) *http.Response {
	header := http.Header{}
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// errReader fails every read, to exercise body reading failures.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }
func (r errReader) Close() error             { return nil }

func BenchmarkBlob(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestBlobSupports(&testing.T{})
		TestBlobDownload(&testing.T{})
	}
}

func TestBlobSupports(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return blobResponse(200, mimeJPEG, ""), nil
	})

	assert.True(t, blob{}.supports(context.TODO(), "http://davidepucci.it"))
}

func TestBlobSupportsError(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("ko")
	})

	assert.False(t, blob{}.supports(context.TODO(), "http://davidepucci.it"))
}

func TestBlobSupportsNotFound(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return blobResponse(404, "", ""), nil
	})

	assert.False(t, blob{}.supports(context.TODO(), "http://davidepucci.it"))
}

func TestBlobSupportsAudioMPEG(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return blobResponse(200, "audio/mpeg", ""), nil
	})

	assert.True(t, blob{}.supports(context.TODO(), "http://davidepucci.it"))
}

func TestBlobUnsupported(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return blobResponse(200, "text/plain", ""), nil
	})

	assert.False(t, blob{}.supports(context.TODO(), "http://davidepucci.it"))
}

func TestBlobDownload(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return blobResponse(200, mimeJPEG, "bitch"), nil
	})

	path := filepath.Join(t.TempDir(), "blob.jpg")
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, blob{}.download(context.TODO(), "http://davidepucci.it", path, stubProcessor(true, nil), ch))
	assert.Equal(t, []byte("bitch"), <-ch)
	content, err := os.ReadFile(path)
	assert.Nil(t, err)
	assert.Equal(t, "bitch", string(content))
}

func TestBlobDownloadProcessorFailure(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return blobResponse(200, mimeJPEG, "bitch"), nil
	})

	path := filepath.Join(t.TempDir(), "blob.jpg")
	assert.EqualError(t, blob{}.download(context.TODO(), "http://davidepucci.it", path, stubProcessor(true, errors.New("ko"))), "ko")
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestBlobDownloadFailure(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("ko")
	})

	// the client wraps transport errors in a url.Error
	assert.ErrorContains(t, blob{}.download(context.TODO(), "http://davidepucci.it", "/dev/null", nil), "ko")
}

func TestBlobDownloadNotFound(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return blobResponse(404, "", ""), nil
	})

	assert.NotNil(t, blob{}.download(context.TODO(), "http://davidepucci.it", "/dev/null", nil))
}

func TestBlobDownloadFileCreationFailure(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return blobResponse(200, mimeJPEG, ""), nil
	})

	// a regular file standing where the parent directory should be
	// makes os.Create fail for real
	blocker := filepath.Join(t.TempDir(), "blocker")
	assert.Nil(t, os.WriteFile(blocker, []byte("x"), 0o600))
	assert.NotNil(t, blob{}.download(context.TODO(), "http://davidepucci.it", filepath.Join(blocker, "blob.jpg"), nil))
}

func TestBlobDownloadReadFailure(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		response := blobResponse(200, mimeJPEG, "")
		response.Body = errReader{errors.New("ko")}
		return response, nil
	})

	path := filepath.Join(t.TempDir(), "blob.jpg")
	assert.EqualError(t, blob{}.download(context.TODO(), "http://davidepucci.it", path, nil), "ko")
}

func TestBlobDownloadProcessorNotApplicable(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return blobResponse(200, mimeJPEG, "data"), nil
	})

	path := filepath.Join(t.TempDir(), "blob.jpg")
	assert.Nil(t, blob{}.download(context.TODO(), "http://davidepucci.it", path, stubProcessor(false, nil)))
	content, err := os.ReadFile(path)
	assert.Nil(t, err)
	assert.Equal(t, "data", string(content))
}

func TestBlobDownloadWriteFailure(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return blobResponse(200, mimeJPEG, "data"), nil
	})

	// a zero file-size rlimit makes the staged write fail for real
	var original syscall.Rlimit
	assert.Nil(t, syscall.Getrlimit(syscall.RLIMIT_FSIZE, &original))
	assert.Nil(t, syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: 0, Max: original.Max}))
	defer func() { assert.Nil(t, syscall.Setrlimit(syscall.RLIMIT_FSIZE, &original)) }()

	path := filepath.Join(t.TempDir(), "blob.jpg")
	assert.NotNil(t, blob{}.download(context.TODO(), "http://davidepucci.it", path, nil))
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestBlobSupportsInvalidURL(t *testing.T) {
	// a malformed URL fails request building before any network call
	assert.False(t, blob{}.supports(context.TODO(), "http://davidepucci.it/\x7f"))
}

func TestBlobDownloadInvalidURL(t *testing.T) {
	// a malformed URL fails request building before any network call
	assert.NotNil(t, blob{}.download(context.TODO(), "http://davidepucci.it/\x7f", "/dev/null", nil))
}

func TestBlobDownloadRemovesPartial(t *testing.T) {
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		response := blobResponse(200, mimeJPEG, "")
		response.Body = errReader{errors.New("ko")}
		return response, nil
	})

	// a failed download leaves no partial file behind
	path := filepath.Join(t.TempDir(), "partial.jpg")
	assert.EqualError(t, blob{}.download(context.TODO(), "http://davidepucci.it", path, nil), "ko")
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}
