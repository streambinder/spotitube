package entity

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/gosimple/slug"
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
	trackId   string
	artworkId string
}

const (
	trackFormat   = "mp3"
	artworkFormat = "jpg"
	lyricsFormat  = "txt"
)

func (track *Track) Path() trackPath {
	return trackPath{
		trackId:   slug.Make(track.ID),
		artworkId: slug.Make(path.Base(track.ArtworkURL)),
	}
}

func (trackPath trackPath) Download() string {
	basename := fmt.Sprintf("%s.%s", trackPath.trackId, trackFormat)
	return util.ErrWrap(filepath.Join("tmp", basename))(
		xdg.CacheFile(filepath.Join("spotitube", basename)))
}

func (trackPath trackPath) Artwork() string {
	basename := fmt.Sprintf("%s.%s", trackPath.artworkId, artworkFormat)
	return util.ErrWrap(filepath.Join("tmp", basename))(
		xdg.CacheFile(filepath.Join("spotitube", basename)))
}

func (trackPath trackPath) Lyrics() string {
	basename := fmt.Sprintf("%s.%s", trackPath.trackId, lyricsFormat)
	return util.ErrWrap(filepath.Join("tmp", basename))(
		xdg.CacheFile(filepath.Join("spotitube", basename)))
}
