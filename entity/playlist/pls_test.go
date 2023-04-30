package playlist

import (
	"io/fs"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestPLS(t *testing.T) {
	var output []byte

	// monkey patching
	defer gomonkey.ApplyFunc(os.WriteFile, func(_ string, data []byte, _ fs.FileMode) error {
		output = data
		return nil
	}).Reset()

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
}
