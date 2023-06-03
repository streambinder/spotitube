package spotify

import (
	"context"
	"strconv"
	"strings"

	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
	"github.com/zmb3/spotify/v2"
)

const TypeTrack = spotify.SearchTypeTrack

func trackEntity(track *spotify.FullTrack) *entity.Track {
	return &entity.Track{
		ID:    track.ID.String(),
		Title: track.Name,
		Artists: func(artists []spotify.SimpleArtist) (flatArtists []string) {
			for _, artist := range artists {
				flatArtists = append(flatArtists, artist.Name)
			}
			return
		}(track.Artists),
		Album: track.Album.Name,
		Artwork: entity.Artwork{
			URL: func(artworks []spotify.Image) string {
				for _, artwork := range artworks {
					return artwork.URL
				}
				return ""
			}(track.Album.Images),
			Data: []byte{},
		},
		Duration:    track.Duration / 1000,
		Lyrics:      "",
		Number:      track.TrackNumber,
		Year:        util.ErrWrap(0)(strconv.Atoi(strings.Split(track.Album.ReleaseDate, "-")[0])),
		UpstreamURL: "",
	}
}

func (client *Client) Track(target string, channels ...chan interface{}) (*entity.Track, error) {
	fullTrack, err := client.GetTrack(context.Background(), id(target))
	if err != nil {
		return nil, err
	}
	track := trackEntity(fullTrack)

	for _, ch := range channels {
		ch <- track
	}
	return track, nil
}
