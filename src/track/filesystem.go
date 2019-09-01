package track

import (
	"fmt"
	"strings"

	"../spotitube"
	"../system"

	"github.com/gosimple/slug"
)

var (
	// JunkSuffixes : array containing every file suffix considered junk
	JunkSuffixes = []string{".ytdl", ".webm", ".opus", ".part", ".jpg", ".tmp", "-id3v2"}
)

// FlushLocal : recheck - and eventually update it - if track is local
func (track Track) FlushLocal() Track {
	if system.FileExists(track.Filename()) {
		track.Local = true
	}

	return track
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
	return fmt.Sprintf("%s.%s", track.Basename(), spotitube.SongExtension)
}

// FilenameTemporary : return Track temporary filename
func (track Track) FilenameTemporary() string {
	return fmt.Sprintf(".%s.%s", slug.Make(track.Basename()), spotitube.SongExtension)
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
