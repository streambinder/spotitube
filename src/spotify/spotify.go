package spotify

import (
	"fmt"
	"strings"

	"../track"

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
func (c *Client) LibraryTracks() ([]*track.Track, error) {
	var (
		tracks     []*track.Track
		iterations int
		options    = defaultOptions()
	)
	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := c.CurrentUsersTracksOpt(&options)
		if err != nil {
			return []*track.Track{}, fmt.Errorf(fmt.Sprintf("Error in %dth chunk of tracks: %s.", iterations, err.Error()))
		}

		for _, t := range chunk.Tracks {
			tAlbum, err := c.Album(t.FullTrack.Album.ID)
			if err != nil {
				tAlbum = &Album{}
			}

			tracks = append(tracks, track.ParseSpotifyTrack(&t.FullTrack, tAlbum))
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
			return fmt.Errorf(fmt.Sprintf("Error removing %dth chunk of removing tracks: %s.", iterations, err.Error()))
		}

		if len(chunk) < 50 {
			break
		}

		iterations++
	}
	return nil
}

// Playlist : return Spotify FullPlaylist from input string playlistURI
func (c *Client) Playlist(uri string) (*Playlist, error) {
	return c.GetPlaylist(IDFromURI(uri))
}

// PlaylistTracks : return array of Spotify FullTrack of all input string playlistURI identified playlist
func (c *Client) PlaylistTracks(uri string) ([]*track.Track, error) {
	var (
		tracks     []*track.Track
		iterations int
		options    = defaultOptions()
	)

	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := c.GetPlaylistTracksOpt(IDFromURI(uri), &options, "")
		if err != nil {
			return []*track.Track{}, fmt.Errorf(fmt.Sprintf("Error reading %dth chunk of tracks: %s.", iterations, err.Error()))
		}

		for _, t := range chunk.Tracks {
			if t.IsLocal {
				continue
			}

			tAlbum, err := c.Album(t.Track.Album.ID)
			if err != nil {
				tAlbum = &Album{}
			}

			tracks = append(tracks, track.ParseSpotifyTrack(&t.Track, tAlbum))
		}

		if len(chunk.Tracks) < 50 {
			break
		}

		iterations++
	}
	return tracks, nil
}

// RemovePlaylistTracks : remove an array of tracks by their IDs from playlist
func (c *Client) RemovePlaylistTracks(uri string, ids []ID) error {
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
		if _, err := c.RemoveTracksFromPlaylist(IDFromURI(uri), chunk...); err != nil {
			return fmt.Errorf(fmt.Sprintf("Error removing %dth chunk of removing tracks: %s.", iterations, err.Error()))
		}

		if len(chunk) < 50 {
			break
		}

		iterations++
	}
	return nil
}

// Album returns a Spotify FullAlbum, specular to the array of Spotify ID
func (c *Client) Album(id ID) (*Album, error) {
	return c.GetAlbum(id)
}

// AlbumTracks returns the array of tracks contained in it
func (c *Client) AlbumTracks(uri string) ([]*track.Track, error) {
	var (
		tracks     []*track.Track
		iterations int
		options    = defaultOptions()
	)

	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := c.GetAlbumTracksOpt(IDFromURI(uri), *options.Limit, *options.Offset)
		if err != nil {
			return []*track.Track{}, fmt.Errorf(fmt.Sprintf("Error reading %dth chunk of tracks: %s.", iterations, err.Error()))
		}

		for _, t := range chunk.Tracks {
			t, err := c.GetTrack(t.ID)
			if err != nil {
				continue
			}

			tAlbum, err := c.Album(t.Album.ID)
			if err != nil {
				tAlbum = &Album{}
			}

			tracks = append(tracks, track.ParseSpotifyTrack(t, tAlbum))
		}

		if len(chunk.Tracks) < 50 {
			break
		}

		iterations++
	}
	return tracks, nil
}

// IDFromURI return a Spotify ID from URI string
func IDFromURI(uri string) ID {
	if strings.Count(uri, ":") == 0 {
		return ID(uri)
	}

	parts := strings.Split(uri, ":")
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
