package spotify

import (
	"github.com/streambinder/spotitube/entity"
	"github.com/zmb3/spotify"
)

func playlistEntity(playlist *spotify.FullPlaylist) *entity.Playlist {
	return &entity.Playlist{
		ID:    playlist.ID.String(),
		Name:  playlist.Name,
		Owner: playlist.Owner.ID,
	}
}

func (client *Client) Playlist(id string, channels ...chan *entity.Track) (*entity.Playlist, error) {
	fullPlaylist, err := client.GetPlaylist(spotify.ID(id))
	if wrapThrottling(err) == errThrottle {
		return client.Playlist(id)
	} else if err != nil {
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
		if err := wrapThrottling(client.NextPage(&fullPlaylist.Tracks)); err == spotify.ErrNoMorePages {
			break
		} else if err != errThrottle && err != nil {
			return nil, err
		}
	}

	return playlist, nil
}
