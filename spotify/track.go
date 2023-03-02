package spotify

import (
	"github.com/streambinder/spotitube/entity"
	"github.com/zmb3/spotify"
)

func trackEntity(track *spotify.SimpleTrack) *entity.Track {
	return &entity.Track{
		ID:          track.ID.String(),
		Title:       track.Name,
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
