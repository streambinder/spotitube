package entity

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadPath(t *testing.T) {
	track := &Track{
		ID:         "123",
		ArtworkURL: "http://domain.tld/123",
	}
	assert.Equal(t, track.Path().trackId+"."+trackFormat, path.Base(track.Path().Download()))
	assert.Equal(t, track.Path().artworkId+"."+artworkFormat, path.Base(track.Path().Artwork()))
	assert.Equal(t, track.Path().trackId+"."+lyricsFormat, path.Base(track.Path().Lyrics()))
}
