package entity

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/gosimple/slug"
	"github.com/streambinder/spotitube/util"
)

type Artwork struct {
	URL  string
	Data []byte
}

type Track struct {
	ID          string
	Title       string
	Artists     []string
	Album       string
	Artwork     Artwork
	Duration    int // in seconds
	Lyrics      string
	Number      int // track number within the album
	Year        int
	UpstreamURL string // URL to the upstream blob the song's been downloaded from
}

type trackPath struct {
	track *Track
}

const (
	TrackFormat   = "mp3"
	ArtworkFormat = "jpg"
	LyricsFormat  = "txt"
)

func (track *Track) Path() trackPath {
	return trackPath{track}
}

func (trackPath trackPath) Final() string {
	return fmt.Sprintf("%s - %s.%s", trackPath.track.Artists[0], trackPath.track.Title, TrackFormat)
}

func (trackPath trackPath) Download() string {
	basename := fmt.Sprintf("%s.%s", slug.Make(trackPath.track.ID), TrackFormat)
	return util.ErrWrap(filepath.Join("tmp", basename))(
		xdg.CacheFile(filepath.Join("spotitube", basename)))
}

func (trackPath trackPath) Artwork() string {
	basename := fmt.Sprintf("%s.%s", slug.Make(path.Base(trackPath.track.Artwork.URL)), ArtworkFormat)
	return util.ErrWrap(filepath.Join("tmp", basename))(
		xdg.CacheFile(filepath.Join("spotitube", basename)))
}

func (trackPath trackPath) Lyrics() string {
	basename := fmt.Sprintf("%s.%s", slug.Make(trackPath.track.ID), LyricsFormat)
	return util.ErrWrap(filepath.Join("tmp", basename))(
		xdg.CacheFile(filepath.Join("spotitube", basename)))
}
