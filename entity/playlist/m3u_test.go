package playlist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkM3U(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestM3U(&testing.T{})
	}
}

func TestM3U(t *testing.T) {
	// testing: the encoder writes to a file named after the playlist in
	// the current directory, so run the test in a temporary one
	dir := t.TempDir()
	t.Chdir(dir)

	encoder := &M3UEncoder{}
	assert.Nil(t, encoder.init(testPlaylist.Name))
	assert.Nil(t, encoder.Add(testTrack))
	assert.Nil(t, encoder.Close())
	assert.Equal(t, "Playlist.m3u", encoder.target)

	output, err := os.ReadFile(filepath.Join(dir, "Playlist.m3u"))
	assert.Nil(t, err)
	assert.Equal(t, `#EXTM3U
#PLAYLIST:Playlist
#EXTINF:0,Artist - Title
Artist - Title.mp3
`, string(output))
}

func TestM3UCloseFailure(t *testing.T) {
	// testing: when the target path cannot be written, Close fails
	dir := t.TempDir()
	t.Chdir(dir)
	assert.Nil(t, os.Mkdir(filepath.Join(dir, "Playlist.m3u"), 0o755))

	encoder := &M3UEncoder{}
	assert.Nil(t, encoder.init(testPlaylist.Name))
	assert.Nil(t, encoder.Add(testTrack))
	assert.Error(t, encoder.Close())
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
