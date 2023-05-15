package entity

import (
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkTrack(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestDownloadPath(&testing.T{})
	}
}

func TestDownloadPath(t *testing.T) {
	track := &Track{
		ID:      "123",
		Artists: []string{"Artist"},
		Artwork: Artwork{URL: "http://domain.tld/123"},
	}
	assert.Equal(t,
		fmt.Sprintf("%s - %s.%s", track.Path().track.Artists[0], track.Path().track.Title, TrackFormat),
		path.Base(track.Path().Final()))
	assert.Equal(t,
		fmt.Sprintf("%s.%s", track.Path().track.ID, TrackFormat),
		path.Base(track.Path().Download()))
	assert.Equal(t,
		fmt.Sprintf("%s.%s", path.Base(track.Path().track.Artwork.URL), ArtworkFormat),
		path.Base(track.Path().Artwork()))
	assert.Equal(t,
		fmt.Sprintf("%s.%s", track.Path().track.ID, LyricsFormat),
		path.Base(track.Path().Lyrics()))
}
