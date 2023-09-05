package playlist

import (
	"io/fs"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
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
	defer gomonkey.ApplyFunc(os.WriteFile, func(_ string, data []byte, _ fs.FileMode) error {
		output = data
		return nil
	}).Reset()

	// testing
	encoder := &M3UEncoder{}
	assert.Nil(t, encoder.init(testPlaylist.Name))
	assert.Nil(t, encoder.Add(testTrack))
	assert.Nil(t, encoder.Close())
	assert.Equal(t, `#EXTM3U
#PLAYLIST:Playlist
#EXTINF:0,Artist - Title
Artist - Title.mp3
`, string(output))
}
