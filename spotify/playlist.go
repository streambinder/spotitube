package spotify

import (
	"context"
	"errors"

	"github.com/streambinder/spotitube/entity/playlist"
	"github.com/zmb3/spotify/v2"
)

func playlistEntity(fullPlaylist *spotify.FullPlaylist) *playlist.Playlist {
	return &playlist.Playlist{
		ID:    fullPlaylist.ID.String(),
		Name:  fullPlaylist.Name,
		Owner: fullPlaylist.Owner.ID,
	}
}

func (client *Client) Playlist(target string, channels ...chan interface{}) (*playlist.Playlist, error) {
	var (
		ctx               = context.Background()
		fullPlaylist, err = client.GetPlaylist(ctx, id(target))
	)
	if err != nil {
		return nil, err
	}

	playlist := playlistEntity(fullPlaylist)
	for {
		for _, playlistTrack := range fullPlaylist.Tracks.Tracks {
			track := trackEntity(&playlistTrack.Track)
			playlist.Tracks = append(playlist.Tracks, track)
			for _, ch := range channels {
				ch <- track
			}
		}

		if err := client.NextPage(ctx, &fullPlaylist.Tracks); errors.Is(err, spotify.ErrNoMorePages) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return playlist, nil
}
