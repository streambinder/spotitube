package playlist

import (
	"io/fs"
	"os"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
)

func BenchmarkPLS(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestPLS(&testing.T{})
	}
}

func TestPLS(t *testing.T) {
	var output []byte

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.WriteFile).To(func(_ string, data []byte, _ fs.FileMode) error {
		output = data
		return nil
	}).Build()

	// testing
	encoder := &PLSEncoder{}
	assert.Nil(t, encoder.init(testPlaylist.Name))
	assert.Nil(t, encoder.Add(testTrack))
	assert.Nil(t, encoder.Close())
	assert.Equal(t, `[Playlist]

File1=Artist - Title.mp3
Title1=Artist - Title
Length1=0

NumberOfEntries=1
`, string(output))
	assert.Equal(t, "Playlist.pls", encoder.target)
}

func TestPLSTargetFilename(t *testing.T) {
	encoder := &PLSEncoder{}
	assert.Nil(t, encoder.init("pop+"))
	assert.Equal(t, "pop+.pls", encoder.target)
	assert.Nil(t, encoder.init("pop-"))
	assert.Equal(t, "pop-.pls", encoder.target)
	assert.Nil(t, encoder.init(`a/b:c`))
	assert.Equal(t, "abc.pls", encoder.target)
}
