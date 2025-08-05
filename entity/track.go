package entity

import (
	"fmt"
	"path"
	"strings"

	"github.com/gosimple/slug"
	"github.com/streambinder/spotitube/sys"
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

type TrackPath struct {
	track *Track
}

const (
	TrackFormat   = "mp3"
	ArtworkFormat = "jpg"
	LyricsFormat  = "txt"
)

// certain track titles include the variant description,
// this functions aims to strip out that part:
// > Title: Name - Acoustic
// > Song:  Name
func (track *Track) Song() (song string) {
	// it can very easily happen to encounter tracks
	// that contains artifacts in the title which do not
	// really define them as songs, rather indicate
	// the variant of the song
	song = track.Title
	song = strings.Split(song+" - ", " - ")[0]
	song = strings.Split(song+" (", " (")[0]
	song = strings.Split(song+" [", " [")[0]
	return
}

func (track *Track) Path() TrackPath {
	return TrackPath{track}
}

func (trackPath TrackPath) Final() string {
	return sys.LegalizeFilename(fmt.Sprintf("%s - %s.%s", trackPath.track.Artists[0], trackPath.track.Title, TrackFormat))
}

func (trackPath TrackPath) Download() string {
	return sys.CacheFile(
		sys.LegalizeFilename(fmt.Sprintf("%s.%s", slug.Make(trackPath.track.ID), TrackFormat)),
	)
}

func (trackPath TrackPath) Artwork() string {
	return sys.CacheFile(
		sys.LegalizeFilename(fmt.Sprintf("%s.%s", slug.Make(path.Base(trackPath.track.Artwork.URL)), ArtworkFormat)),
	)
}

func (trackPath TrackPath) Lyrics() string {
	return sys.CacheFile(
		sys.LegalizeFilename(fmt.Sprintf("%s.%s", slug.Make(trackPath.track.ID), LyricsFormat)),
	)
}
