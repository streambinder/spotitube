package spotify

import (
	"context"
	"errors"

	"github.com/streambinder/spotitube/entity"
	"github.com/zmb3/spotify/v2"
)

func playlistEntity(playlist *spotify.FullPlaylist) *entity.Playlist {
	return &entity.Playlist{
		ID:    playlist.ID.String(),
		Name:  playlist.Name,
		Owner: playlist.Owner.ID,
	}
}

func (client *Client) Playlist(id string, channels ...chan interface{}) (*entity.Playlist, error) {
	var (
		ctx               = context.Background()
		fullPlaylist, err = client.GetPlaylist(ctx, spotify.ID(id))
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
