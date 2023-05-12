package spotify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify/v2"
)

func TestID(t *testing.T) {
	var (
		target    = "1234567890123456789012"
		spotifyId = spotify.ID(target)
	)
	assert.Equal(t, id(target), spotifyId)
	assert.Equal(t, id("spotify:track:"+target), spotifyId)
	assert.Equal(t, id("https://open.spotify.com/track/"+target), spotifyId)
	assert.Equal(t, id("https://open.spotify.com/track/"+target+"?si=abcdefghijklmnop"), spotifyId)
}
