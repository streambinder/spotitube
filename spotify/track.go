package spotify

import (
	"github.com/streambinder/spotitube/entity"
	"github.com/zmb3/spotify"
)

func trackEntity(fullTrack *spotify.FullTrack) *entity.Track {
	return &entity.Track{
		ID:          fullTrack.ID.String(),
		Title:       fullTrack.Name,
		Artists:     []string{},
		Album:       "",
		ArtworkURL:  "",
		Artwork:     []byte{},
		Duration:    0,
		Genre:       "",
		Lyrics:      []byte{},
		Number:      0,
		Year:        "",
		UpstreamURL: "",
	}
}
