package track

import (
	"fmt"
	"strings"

	"github.com/gosimple/slug"
	"github.com/streambinder/spotitube/src/system"
)

var (
	extension = "mp3"

	// JunkSuffixes : array containing every file suffix considered junk
	JunkSuffixes = []string{".ytdl", ".webm", ".opus", ".part", ".jpg", ".tmp", "-id3v2"}
)

// Local returns a boolean indicating whether track is on filesystem or not
func (track *Track) Local() bool {
	return system.FileExists(track.Filename())
}

// Basename : return track basename
func (track Track) Basename() string {
	basename := track.Artist + " - " + track.Title
	for _, symbol := range []string{"/", "\\", ".", "?", "<", ">", ":", "*"} {
		basename = strings.Replace(basename, symbol, "", -1)
	}
	basename = strings.Replace(basename, "  ", " ", -1)
	basename = system.Asciify(basename)
	return strings.TrimSpace(basename)
}

// Filename : return track filename
func (track Track) Filename() string {
	return fmt.Sprintf("%s.%s", track.Basename(), extension)
}

// FilenameTemporary : return Track temporary filename
func (track Track) FilenameTemporary() string {
	return fmt.Sprintf(".%s.%s", slug.Make(track.Basename()), extension)
}

// FilenameArtwork : return Track artwork filename
func (track Track) FilenameArtwork() string {
	return fmt.Sprintf(".%s.jpg", strings.Split(track.Image, "/")[len(strings.Split(track.Image, "/"))-1])
}

// TempFiles : return strings array containing all possible junk file names
func (track Track) TempFiles() []string {
	var tempFiles []string
	for _, fnamePrefix := range []string{track.FilenameTemporary(), track.FilenameArtwork()} {
		tempFiles = append(tempFiles, fnamePrefix)
		for _, fnameJunkSuffix := range JunkSuffixes {
			tempFiles = append(tempFiles, fnamePrefix+fnameJunkSuffix)
		}
	}
	return tempFiles
}

// JunkWildcards : return strings array containing all possible junk filenames wilcards
func JunkWildcards() []string {
	var junkWildcards []string
	for _, junkSuffix := range JunkSuffixes {
		junkWildcards = append(junkWildcards, ".*"+junkSuffix)
	}
	return append(junkWildcards, ".*.mp3")
}
