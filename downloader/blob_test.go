package downloader

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/bytedance/mockey"
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
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Head")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
	}, nil).Build()

	// testing
	assert.True(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobSupportsError(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Head")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.False(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobSupportsNotFound(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Head")).Return(&http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil).Build()

	// testing
	assert.False(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobUnsupported(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Head")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     map[string][]string{"Content-Type": {"text/plain"}},
	}, nil).Build()

	// testing
	assert.False(t, blob{}.supports("http://davidepucci.it"))
}

func TestBlobDownload(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("bitch")),
		Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
	}, nil).Build()
	mockey.Mock(os.OpenFile).Return(nil, nil).Build()
	mockey.Mock(io.ReadAll).Return([]byte{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&os.File{}, "Write")).Return(0, nil).Build()

	// testing
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, blob{}.download("http://davidepucci.it", "/dev/null", stubProcessor(true, nil), ch))
}

func TestBlobDownloadProcessorFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("bitch")),
		Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
	}, nil).Build()
	mockey.Mock(os.OpenFile).Return(nil, nil).Build()
	mockey.Mock(io.ReadAll).Return([]byte{}, nil).Build()

	// testing
	assert.EqualError(t, blob{}.download("http://davidepucci.it", "/dev/null", stubProcessor(true, errors.New("ko"))), "ko")
}

func TestBlobDownloadFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, blob{}.download("http://davidepucci.it", "/dev/null", nil), "ko")
}

func TestBlobDownloadNotFound(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil).Build()

	// testing
	assert.NotNil(t, blob{}.download("http://davidepucci.it", "/dev/null", nil))
}

func TestBlobDownloadFileCreationFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
	}, nil).Build()
	mockey.Mock(os.OpenFile).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, blob{}.download("http://davidepucci.it", "/dev/null", nil), "ko")
}

func TestBlobDownloadReadFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
	}, nil).Build()
	mockey.Mock(os.OpenFile).Return(nil, nil).Build()
	mockey.Mock(io.ReadAll).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, blob{}.download("http://davidepucci.it", "/dev/null", nil), "ko")
}

func TestBlobDownloadProcessorNotApplicable(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("data")),
		Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
	}, nil).Build()
	mockey.Mock(os.OpenFile).Return(nil, nil).Build()
	mockey.Mock(io.ReadAll).Return([]byte{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&os.File{}, "Write")).Return(0, nil).Build()

	// testing
	assert.Nil(t, blob{}.download("http://davidepucci.it", "/dev/null", stubProcessor(false, nil)))
}

func TestBlobDownloadWriteFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("data")),
		Header:     map[string][]string{"Content-Type": {"image/jpeg"}},
	}, nil).Build()
	mockey.Mock(os.OpenFile).Return(nil, nil).Build()
	mockey.Mock(io.ReadAll).Return([]byte{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&os.File{}, "Write")).Return(0, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, blob{}.download("http://davidepucci.it", "/dev/null", nil), "ko")
}
