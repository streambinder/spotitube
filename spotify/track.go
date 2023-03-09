package spotify

import (
	"strings"

	"github.com/streambinder/spotitube/entity"
	"github.com/zmb3/spotify"
)

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
		ArtworkURL: func(artworks []spotify.Image) string {
			for _, artwork := range artworks {
				return artwork.URL
			}
			return ""
		}(track.Album.Images),
		Artwork:     []byte{},
		Duration:    track.Duration / 1000,
		Lyrics:      []byte{},
		Number:      track.TrackNumber,
		Year:        strings.Split(track.Album.ReleaseDate, "-")[0],
		UpstreamURL: "",
	}
}

func (client *Client) Track(id string, channels ...chan interface{}) (*entity.Track, error) {
	fullTrack, err := client.GetTrack(spotify.ID(id))
	if err != nil {
		return nil, err
	}
	track := trackEntity(fullTrack)

	for _, ch := range channels {
		ch <- track
	}
	return track, nil
}
