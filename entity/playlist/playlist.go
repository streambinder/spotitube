package playlist

import (
	"errors"
	"strings"

	"github.com/streambinder/spotitube/entity"
)

type Playlist struct {
	ID            string
	Name          string
	Owner         string
	Collaborative bool
	Tracks        []*entity.Track
}

func (entity Playlist) Encoder(encoding string) (playlistEncoder, error) {
	var encoder playlistEncoder
	switch strings.ToLower(encoding) {
	case "m3u":
		encoder = &M3UEncoder{}
	case "pls":
		encoder = &PLSEncoder{}
	default:
		return nil, errors.New("unsupported encoding")
	}

	if err := encoder.init(entity.Name); err != nil {
		return nil, err
	}

	return encoder, nil
}
