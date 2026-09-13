package playlist

import (
	"io/fs"
	"os"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
)

func BenchmarkM3U(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestM3U(&testing.T{})
	}
}

func TestM3U(t *testing.T) {
	var output []byte

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.WriteFile).To(func(_ string, data []byte, _ fs.FileMode) error {
		output = data
		return nil
	}).Build()

	// testing
	encoder := &M3UEncoder{}
	assert.Nil(t, encoder.init(testPlaylist.Name))
	assert.Nil(t, encoder.Add(testTrack))
	assert.Nil(t, encoder.Close())
	assert.Equal(t, "Playlist.m3u", encoder.target)
	assert.Equal(t, `#EXTM3U
#PLAYLIST:Playlist
#EXTINF:0,Artist - Title
Artist - Title.mp3
`, string(output))
}

func TestM3UTargetFilename(t *testing.T) {
	encoder := &M3UEncoder{}
	assert.Nil(t, encoder.init("pop+"))
	assert.Equal(t, "pop+.m3u", encoder.target)
	assert.Nil(t, encoder.init("pop-"))
	assert.Equal(t, "pop-.m3u", encoder.target)
	assert.Nil(t, encoder.init(`a/b:c`))
	assert.Equal(t, "abc.m3u", encoder.target)
}
