package playlist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkPLS(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestPLS(&testing.T{})
	}
}

func TestPLS(t *testing.T) {
	// testing: the encoder writes to a file named after the playlist in
	// the current directory, so run the test in a temporary one
	dir := t.TempDir()
	t.Chdir(dir)

	encoder := &PLSEncoder{}
	assert.Nil(t, encoder.init(testPlaylist.Name))
	assert.Nil(t, encoder.Add(testTrack))
	assert.Nil(t, encoder.Close())
	assert.Equal(t, `[Playlist]

File1=Artist - Title.mp3
Title1=Artist - Title
Length1=0

NumberOfEntries=1
`, readFile(t, filepath.Join(dir, "Playlist.pls")))
	assert.Equal(t, "Playlist.pls", encoder.target)
}

func TestPLSCloseFailure(t *testing.T) {
	// testing: when the target path cannot be written, Close fails
	dir := t.TempDir()
	t.Chdir(dir)
	assert.Nil(t, os.Mkdir(filepath.Join(dir, "Playlist.pls"), 0o755))

	encoder := &PLSEncoder{}
	assert.Nil(t, encoder.init(testPlaylist.Name))
	assert.Nil(t, encoder.Add(testTrack))
	assert.Error(t, encoder.Close())
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

// readFile reads a file and returns its contents as a string
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.Nil(t, err)
	return string(data)
}
