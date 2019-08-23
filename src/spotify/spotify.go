package spotify

import (
	"fmt"
	"strings"

	"github.com/thanhpk/randstr"
	"github.com/zmb3/spotify"
)

var (
	clientState         = randstr.Hex(20)
	clientAuthenticator spotify.Authenticator
)

// User : get authenticated username from authenticated client
func (c *Client) User() (string, string) {
	if user, err := c.CurrentUser(); err == nil {
		return user.DisplayName, user.ID
	}
	return "unknown", "unknown"
}

// LibraryTracks : return array of Spotify FullTrack of all authenticated user library songs
func (c *Client) LibraryTracks() ([]Track, error) {
	var (
		tracks     []Track
		iterations int
		options    = defaultOptions()
	)
	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := c.CurrentUsersTracksOpt(&options)
		if err != nil {
			return []Track{}, fmt.Errorf(fmt.Sprintf("Something gone wrong while reading %dth chunk of tracks: %s.", iterations, err.Error()))
		}
		for _, track := range chunk.Tracks {
			tracks = append(tracks, track.FullTrack)
		}
		if len(chunk.Tracks) < 50 {
			break
		}
		iterations++
	}
	return tracks, nil
}

// RemoveLibraryTracks : remove an array of tracks by their IDs from library
func (c *Client) RemoveLibraryTracks(ids []ID) error {
	if len(ids) == 0 {
		return nil
	}

	var iterations int
	for true {
		lowerbound := iterations * 50
		upperbound := lowerbound + 50
		if len(ids) < upperbound {
			upperbound = lowerbound + (len(ids) - lowerbound)
		}
		chunk := ids[lowerbound:upperbound]
		if err := c.RemoveTracksFromLibrary(chunk...); err != nil {
			return fmt.Errorf(fmt.Sprintf("Something gone wrong while removing %dth chunk of removing tracks: %s.", iterations, err.Error()))
		}
		if len(chunk) < 50 {
			break
		}
		iterations++
	}
	return nil
}

// Playlist : return Spotify FullPlaylist from input string playlistURI
func (c *Client) Playlist(playlistURI string) (*Playlist, error) {
	return c.GetPlaylist(PlaylistID(playlistURI))
}

// PlaylistTracks : return array of Spotify FullTrack of all input string playlistURI identified playlist
func (c *Client) PlaylistTracks(playlistURI string) ([]Track, error) {
	var (
		tracks     []Track
		iterations int
		options    = defaultOptions()
	)

	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := c.GetPlaylistTracksOpt(PlaylistID(playlistURI), &options, "")
		if err != nil {
			return []Track{}, fmt.Errorf(fmt.Sprintf("Something gone wrong while reading %dth chunk of tracks: %s.", iterations, err.Error()))
		}
		for _, track := range chunk.Tracks {
			if !track.IsLocal {
				tracks = append(tracks, track.Track)
			}
		}
		if len(chunk.Tracks) < 50 {
			break
		}
		iterations++
	}
	return tracks, nil
}

// RemovePlaylistTracks : remove an array of tracks by their IDs from playlist
func (c *Client) RemovePlaylistTracks(playlistURI string, ids []ID) error {
	if len(ids) == 0 {
		return nil
	}

	var (
		iterations int
	)
	for true {
		lowerbound := iterations * 50
		upperbound := lowerbound + 50
		if len(ids) < upperbound {
			upperbound = lowerbound + (len(ids) - lowerbound)
		}
		chunk := ids[lowerbound:upperbound]
		if _, err := c.RemoveTracksFromPlaylist(PlaylistID(playlistURI), chunk...); err != nil {
			return fmt.Errorf(fmt.Sprintf("Something gone wrong while removing %dth chunk of removing tracks: %s.", iterations, err.Error()))
		}
		if len(chunk) < 50 {
			break
		}
		iterations++
	}
	return nil
}

// Albums : return array Spotify FullAlbum, specular to the array of Spotify ID
func (c *Client) Albums(ids []ID) ([]Album, error) {
	var (
		albums     []spotify.FullAlbum
		iterations int
		upperbound int
		lowerbound int
	)
	for true {
		lowerbound = iterations * 20
		if upperbound = lowerbound + 20; upperbound >= len(ids) {
			upperbound = lowerbound + (len(ids) - lowerbound)
		}
		chunk, err := c.GetAlbums(ids[lowerbound:upperbound]...)
		if err != nil {
			var chunk []spotify.FullAlbum
			for _, albumID := range ids[lowerbound:upperbound] {
				album, err := c.GetAlbum(albumID)
				if err == nil {
					chunk = append(chunk, *album)
				} else {
					chunk = append(chunk, spotify.FullAlbum{})
				}
			}
		}
		for _, album := range chunk {
			albums = append(albums, *album)
		}
		if len(chunk) < 20 {
			break
		}
		iterations++
	}
	return albums, nil
}

// PlaylistID : return a Spotify playlist ID from playlist URI string
func PlaylistID(playlistURI string) ID {
	if strings.Count(playlistURI, ":") == 0 {
		return ID(playlistURI)
	}
	parts := strings.Split(playlistURI, ":")
	return ID(parts[len(parts)-1])
}

func defaultOptions() spotify.Options {
	var (
		optLimit  = 50
		optOffset = 0
	)
	return spotify.Options{
		Limit:  &optLimit,
		Offset: &optOffset,
	}
}
