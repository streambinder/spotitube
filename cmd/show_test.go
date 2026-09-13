package cmd

import (
	"errors"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

func BenchmarkShow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdShow(&testing.T{})
	}
}

func TestCmdShow(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "AttachedPicture")).Return("image/jpeg", []byte("some picture data")).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "Duration")).Return("60").Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdShow(), "path/to/track1", "path/to/track2")))
}

func TestCmdShowOpenFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3.Open).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdShow(), "path/to/track")), "ko")
}

func TestCmdShowPictureFallback(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdShow(), "path/to/track")))
}
