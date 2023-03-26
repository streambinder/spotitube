package entity

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/streambinder/spotitube/util"
)

type Track struct {
	ID          string
	Title       string
	Artists     []string
	Album       string
	ArtworkURL  string // URL whose content to feed the Artwork field with
	Artwork     []byte
	Duration    int // in seconds
	Lyrics      []byte
	Number      int // track number within the album
	Year        string
	UpstreamURL string // URL to the upstream blob the song's been downloaded from
}

type trackPath struct {
	id string
}

const format = "mp3"

func (track *Track) Path() trackPath {
	return trackPath{strings.ToLower(track.ID)}
}

func (trackPath trackPath) Download() string {
	basename := fmt.Sprintf("%s.%s", trackPath.id, format)
	return util.ErrWrap(filepath.Join("tmp", basename))(xdg.CacheFile(filepath.Join("spotitube", basename)))
}
