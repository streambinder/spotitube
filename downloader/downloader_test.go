package downloader

import (
	"errors"
	"io/fs"
	"os"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
)

func TestDownload(t *testing.T) {
	// monkey patching
	monkey.Patch(os.MkdirAll, func(string, fs.FileMode) error { return nil })
	defer monkey.Unpatch(os.MkdirAll)
	monkey.Patch(cmd.YouTubeDl, func(url, path string) error { return nil })
	defer monkey.Unpatch(cmd.YouTubeDl)
	monkey.PatchInstanceMethod(reflect.TypeOf(youTubeDl{}), "Download",
		func(youTubeDl, string, string) error { return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(youTubeDl{}), "Download")

	// testing
	assert.Nil(t, Download("http://youtu.be", "fname.txt"))
}

func TestDownloadMakeDirFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(os.MkdirAll, func(string, fs.FileMode) error { return errors.New("failure") })
	defer monkey.Unpatch(os.MkdirAll)

	// testing
	assert.Error(t, Download("http://youtu.be", "fname.txt"), "failure")
}

func TestDownloadUnsupported(t *testing.T) {
	// monkey patching
	monkey.Patch(cmd.YouTubeDl, func(url, path string) error { return nil })
	defer monkey.Unpatch(cmd.YouTubeDl)

	// testing
	assert.Error(t, Download("http://davidepucci.it", "fname.txt"), "unsupported url")
}
