package playlist

import (
	"github.com/streambinder/pango/entity"
)

type playlistEncoder interface {
	init(string) error
	Add(*entity.Track) error
	Close() error
}
