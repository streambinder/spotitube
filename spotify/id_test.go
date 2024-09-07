package spotify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify/v2"
)

func BenchmarkID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestID(&testing.T{})
	}
}

func TestID(t *testing.T) {
	var (
		target    = "1234567890123456789012"
		spotifyID = spotify.ID(target)
	)
	assert.Equal(t, id(target), spotifyID)
	assert.Equal(t, id("spotify:track:"+target), spotifyID)
	assert.Equal(t, id("https://open.spotify.com/track/"+target), spotifyID)
	assert.Equal(t, id("https://open.spotify.com/track/"+target+"?si=abcdefghijklmnop"), spotifyID)
}
