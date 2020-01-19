package track

import (
	"fmt"
	"strings"

	"github.com/gosimple/slug"
	"github.com/streambinder/spotitube/system"
)

var (
	extension    = "mp3"
	junkSuffixes = []string{".ytdl", ".webm", ".opus", ".part", ".jpg", ".tmp", "-id3v2"}
)

// Local returns a boolean indicating whether track is on filesystem or not
func (track *Track) Local() bool {
	return system.FileExists(track.Filename())
}

// Basename returns track basename
func (track Track) Basename() string {
	basename := track.Artist + " - " + track.Title
	for _, symbol := range []string{"/", "\\", ".", "?", "<", ">", ":", "*"} {
		basename = strings.Replace(basename, symbol, "", -1)
	}
	basename = strings.Replace(basename, "  ", " ", -1)
	basename = system.Asciify(basename)
	return strings.TrimSpace(basename)
}

// Query returns string used to search song online
func (track Track) Query() string {
	return track.Basename()
}

// Filename returns track filename
func (track Track) Filename() string {
	return fmt.Sprintf("%s.%s", track.Basename(), extension)
}

// FilenameTemporary returns track temporary filename
func (track Track) FilenameTemporary() string {
	return fmt.Sprintf(".%s.%s", slug.Make(track.Basename()), extension)
}

// FilenameArtwork returns track artwork filename
func (track Track) FilenameArtwork() string {
	return fmt.Sprintf(".%s.jpg", strings.Split(track.Image, "/")[len(strings.Split(track.Image, "/"))-1])
}

// JunkWildcards returns strings array containing junk filenames wilcards
func JunkWildcards() (wildcards []string) {
	wildcards = append(wildcards, ".*.mp3")
	for _, junkSuffix := range junkSuffixes {
		wildcards = append(wildcards, ".*"+junkSuffix)
	}
	return
}
