package spotify

import (
	api "github.com/zmb3/spotify"
)

// Spotify : struct object containing all the informations needed to authenticate and fetch from Spotify
type Spotify struct {
	Client *api.Client
}
