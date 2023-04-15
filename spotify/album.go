package spotify

import (
	"context"
	"errors"

	"github.com/streambinder/spotitube/entity"
	"github.com/zmb3/spotify/v2"
)

func albumEntity(album *spotify.FullAlbum) *entity.Album {
	return &entity.Album{
		ID:   album.ID.String(),
		Name: album.Name,
		Artists: func(artists []spotify.SimpleArtist) (flatArtists []string) {
			for _, artist := range artists {
				flatArtists = append(flatArtists, artist.Name)
			}
			return
		}(album.Artists),
	}
}

func (client *Client) Album(id string, channels ...chan interface{}) (*entity.Album, error) {
	var (
		ctx            = context.Background()
		fullAlbum, err = client.GetAlbum(ctx, spotify.ID(id))
	)
	if err != nil {
		return nil, err
	}

	album := albumEntity(fullAlbum)
	for {
		for _, albumTrack := range fullAlbum.Tracks.Tracks {
			track := trackEntity(&spotify.FullTrack{
				SimpleTrack: albumTrack,
				Album:       fullAlbum.SimpleAlbum,
			})
			album.Tracks = append(album.Tracks, track)
			for _, ch := range channels {
				ch <- track
			}
		}

		if err := client.NextPage(ctx, &fullAlbum.Tracks); errors.Is(err, spotify.ErrNoMorePages) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return album, nil
}
