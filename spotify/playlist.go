package spotify

import (
	"context"
	"errors"

	"github.com/gosimple/slug"
	"github.com/streambinder/spotitube/entity/playlist"
	"github.com/zmb3/spotify/v2"
)

const (
	personalPlaylistsCacheID = "PersonalPlaylists"
)

func playlistEntity(fullPlaylist *spotify.FullPlaylist) *playlist.Playlist {
	return &playlist.Playlist{
		ID:    fullPlaylist.ID.String(),
		Name:  fullPlaylist.Name,
		Owner: fullPlaylist.Owner.ID,
	}
}

func (client *Client) personalPlaylistNameToID(target string) (spotify.ID, error) {
	playlistsMap, ok := client.cache[personalPlaylistsCacheID]
	if !ok {
		playlistsMap = make(map[string]string)
		personalPlaylists, err := client.personalPlaylists()
		if err != nil {
			return "", err
		}
		for _, playlist := range personalPlaylists {
			playlistsMap.(map[string]string)[slug.Make(playlist.Name)] = playlist.ID
		}
		client.cache[personalPlaylistsCacheID] = playlistsMap
	}

	if cachedTarget, ok := playlistsMap.(map[string]string)[slug.Make(target)]; ok {
		return id(cachedTarget), nil
	}
	return id(target), nil
}

func (client *Client) personalPlaylists() ([]*playlist.Playlist, error) {
	var (
		ctx                = context.Background()
		userPlaylists, err = client.CurrentUsersPlaylists(ctx)
		playlists          []*playlist.Playlist
	)
	if err != nil {
		return nil, err
	}

	for {
		for _, playlist := range userPlaylists.Playlists {
			playlists = append(playlists, playlistEntity(&spotify.FullPlaylist{SimplePlaylist: playlist}))
		}

		if err := client.NextPage(ctx, userPlaylists); errors.Is(err, spotify.ErrNoMorePages) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return playlists, nil
}

func (client *Client) Playlist(target string, channels ...chan interface{}) (*playlist.Playlist, error) {
	var (
		ctx     = context.Background()
		id, err = client.personalPlaylistNameToID(target)
	)
	if err != nil {
		return nil, err
	}

	fullPlaylist, err := client.GetPlaylist(ctx, id)
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
