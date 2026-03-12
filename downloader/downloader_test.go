package downloader

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/sys/cmd"
	"github.com/stretchr/testify/assert"
)

func BenchmarkDownload(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestDownload(&testing.T{})
	}
}

func TestDownload(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("not exists")).Build()
	mockey.Mock(os.MkdirAll).Return(nil).Build()
	mockey.Mock(cmd.YouTubeDl).Return(nil).Build()

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
	defer mockey.UnPatchAll()
	mockey.Mock(os.ReadFile).Return([]byte{}, nil).Build()

	// testing
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, Download("http://youtu.be", "fname.txt", nil, ch))
}

func TestDownloadMakeDirFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("not exists")).Build()
	mockey.Mock(os.MkdirAll).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, Download("http://youtu.be", "fname.txt", nil), "ko")
}

func TestDownloadUnsupported(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("not exists")).Build()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Head")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     map[string][]string{"Content-Type": {"text/plain"}},
	}, nil).Build()

	// testing
	assert.Error(t, Download("http://davidepucci.it", "fname.txt", nil))
}

func TestDownloadYouTubeDlFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("not exists")).Build()
	mockey.Mock(os.MkdirAll).Return(nil).Build()
	mockey.Mock(cmd.YouTubeDl).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, Download("http://youtu.be", "fname.txt", nil), "ko")
}
