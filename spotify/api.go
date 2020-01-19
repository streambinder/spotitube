package spotify

import (
	"fmt"
	"os"

	"github.com/zmb3/spotify"
)

// Playlist is an alias for Spotify FullPlaylist
type Playlist = spotify.FullPlaylist

// Album is an alias for Spotify FullAlbum
type Album = spotify.FullAlbum

// Track is an alias for Spotify FullTrack
type Track = spotify.FullTrack

// ID is an alias for Spotify ID
type ID = spotify.ID

const (
	clientID           = ""
	clientIDEnvKey     = "SPOTIFY_ID"
	clientSecret       = ""
	clientSecretEnvKey = "SPOTIFY_KEY"
)

// Ready returns an error if further configurations
// are needed to access Spotify Apis
func Ready() error {
	if len(clientID) != 32 && len(os.Getenv(clientIDEnvKey)) != 32 {
		return fmt.Errorf(clientIDEnvKey + " not found")
	}

	if len(clientSecret) != 32 && len(os.Getenv(clientSecretEnvKey)) != 32 {
		return fmt.Errorf(clientSecretEnvKey + " not found")
	}

	return nil
}
