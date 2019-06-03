package spotify

import "github.com/zmb3/spotify"

// Playlist : alias for Spotify FullPlaylist
type Playlist = spotify.FullPlaylist

// Album : alias for Spotify FullAlbum
type Album = spotify.FullAlbum

// Track : alias for Spotify FullTrack
type Track = spotify.FullTrack

// ID : alias for Spotify ID
type ID = spotify.ID

const (
	// SpotifyClientID : Spotify app client ID
	SpotifyClientID = ":SPOTIFY_CLIENT_ID:"
	// SpotifyClientSecret : Spotify app client secret key
	SpotifyClientSecret = ":SPOTIFY_CLIENT_SECRET:"
)
