package spotify

import (
	"fmt"
	"strings"
	"time"

	"github.com/streambinder/spotitube/track"
	"github.com/thanhpk/randstr"
	"github.com/zmb3/spotify"
)

const (
	// RateLimitDelay is the amount of time before
	// retrying with an API request that have
	// been blocked due to API rate limit
	RateLimitDelay = 5 * time.Second
)

var (
	clientState         = randstr.Hex(20)
	clientAuthenticator spotify.Authenticator

	rateLimitFactor = 1
)

// User returns session authenticated user
func (c *Client) User() (string, string) {
	if user, err := c.CurrentUser(); err == nil {
		return user.DisplayName, user.ID
	}

	return "unknown", "unknown"
}

// LibraryTracks returns library tracks
func (c *Client) LibraryTracks() ([]*track.Track, error) {
	var (
		tracks     []*track.Track
		iterations int
		options    = defaultOptions()
	)
	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := c.CurrentUsersTracksOpt(&options)
		if err != nil && err.Error() == "API rate limit exceeded." {
			time.Sleep(time.Duration(rateLimitFactor) * RateLimitDelay)
			rateLimitFactor++
			continue
		} else if err != nil {
			return nil, err
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

// RemoveLibraryTracks removes an array of tracks by their IDs from library
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
			return err
		}

		if len(chunk) < 50 {
			break
		}

		iterations++
	}
	return nil
}

// Playlist returns a Playlist object from given URI
func (c *Client) Playlist(uri string) (*Playlist, error) {
	return c.GetPlaylist(IDFromURI(uri))
}

// PlaylistTracks returns playlist tracks from given URI
func (c *Client) PlaylistTracks(uri string) ([]*track.Track, error) {
	var (
		tracks     []*track.Track
		iterations int
		options    = defaultOptions()
	)

	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := c.GetPlaylistTracksOpt(IDFromURI(uri), &options, "")
		if err != nil && err.Error() == "API rate limit exceeded." {
			fmt.Println("entered")
			time.Sleep(time.Duration(rateLimitFactor) * RateLimitDelay)
			rateLimitFactor++
			continue
		} else if err != nil {
			return nil, err
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

// RemovePlaylistTracks removes an array of tracks by their IDs from playlist
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
			return err
		}

		if len(chunk) < 50 {
			break
		}

		iterations++
	}
	return nil
}

// Album returns a Album object from given URI
func (c *Client) Album(id ID) (*Album, error) {
	return c.GetAlbum(id)
}

// AlbumTracks returns album tracks from given URI
func (c *Client) AlbumTracks(uri string) ([]*track.Track, error) {
	var (
		tracks     []*track.Track
		iterations int
		options    = defaultOptions()
	)

	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := c.GetAlbumTracksOpt(IDFromURI(uri), *options.Limit, *options.Offset)
		if err != nil && err.Error() == "API rate limit exceeded." {
			time.Sleep(time.Duration(rateLimitFactor) * RateLimitDelay)
			rateLimitFactor++
			continue
		} else if err != nil {
			return nil, err
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

// IDFromURI returns a Spotify ID from URI string
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
