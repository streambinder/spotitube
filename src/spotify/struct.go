package spotify

import (
	api "github.com/zmb3/spotify"
)

// Spotify : struct object containing all the informations needed to authenticate and fetch from Spotify
type Spotify struct {
	Client *api.Client
}

// SpotifyAuthURL : struct object containing both the full authentication URL provided by Spotify and the shortened one using TinyURL
type SpotifyAuthURL struct {
	Full  string
	Short string
}
