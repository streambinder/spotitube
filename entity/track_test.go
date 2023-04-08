package entity

import (
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadPath(t *testing.T) {
	track := &Track{
		ID:         "123",
		Artists:    []string{"Artist"},
		ArtworkURL: "http://domain.tld/123",
	}
	assert.Equal(t,
		fmt.Sprintf("%s - %s.%s", track.Path().track.Artists[0], track.Path().track.Title, trackFormat),
		path.Base(track.Path().Final()))
	assert.Equal(t,
		fmt.Sprintf("%s.%s", track.Path().track.ID, trackFormat),
		path.Base(track.Path().Download()))
	assert.Equal(t,
		fmt.Sprintf("%s.%s", path.Base(track.Path().track.ArtworkURL), artworkFormat),
		path.Base(track.Path().Artwork()))
	assert.Equal(t,
		fmt.Sprintf("%s.%s", track.Path().track.ID, lyricsFormat),
		path.Base(track.Path().Lyrics()))
}
