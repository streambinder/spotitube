package track

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gosimple/slug"
)

const (
	// SongTypeAlbum : identifier for Song in its album variant
	SongTypeAlbum = iota
	// SongTypeLive : identifier for Song in its live variant
	SongTypeLive
	// SongTypeCover : identifier for Song in its cover variant
	SongTypeCover
	// SongTypeRemix : identifier for Song in its remix variant
	SongTypeRemix
	// SongTypeAcoustic : identifier for Song in its acoustic variant
	SongTypeAcoustic
	// SongTypeKaraoke : identifier for Song in its karaoke variant
	SongTypeKaraoke
	// SongTypeParody : identifier for Song in its parody variant
	SongTypeParody
	// SongTypeReverse : identifier for Song in its reverse variant
	SongTypeReverse
)

var (
	// SongTypes : array containing every song variant identifier
	SongTypes = []int{SongTypeLive, SongTypeCover, SongTypeRemix,
		SongTypeAcoustic, SongTypeKaraoke, SongTypeParody}
)

// Type : return track variant
func (track Track) Type() int {
	for _, songType := range SongTypes {
		if IsType(track.Title, songType) {
			return songType
		}
	}
	return SongTypeAlbum
}

// IsType : return True if input sequence matches with selected input songType variant
func IsType(sequence string, songType int) bool {
	var regexes []string
	if songType == SongTypeLive {
		regexes = []string{slug.Make("@"), slug.Make("live"), slug.Make("perform"), slug.Make("tour"), "[1-2]{1}[0-9]{3}"}
	} else if songType == SongTypeCover {
		regexes = []string{slug.Make("cover"), slug.Make("vs"), slug.Make("amateur")}
	} else if songType == SongTypeRemix {
		regexes = []string{slug.Make("remix"), slug.Make("radio-edit")}
	} else if songType == SongTypeAcoustic {
		regexes = []string{slug.Make("acoustic")}
	} else if songType == SongTypeKaraoke {
		regexes = []string{slug.Make("karaoke"), slug.Make("instrumental")}
	} else if songType == SongTypeParody {
		regexes = []string{slug.Make("parody")}
	} else if songType == SongTypeReverse {
		regexes = []string{slug.Make("reverse")}
	}

	match, _ := regexp.MatchString(fmt.Sprintf(`(\A|-)(%s)(-|\z)`, strings.Join(regexes, "|")), slug.Make(sequence))
	return match
}

// Seems : return nil error if sequence is input sequence string matches with Track
func (track Track) Seems(sequence string) error {
	if err := track.SeemsByWordMatch(sequence); err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(sequence), "full album") {
		return fmt.Errorf("Item seems to be pointing to an album, not to a song")
	}
	for _, songType := range SongTypes {
		if IsType(sequence, songType) && track.Type() != songType {
			return fmt.Errorf("Songs seem to be of different types")
		}
	}
	return nil
}

// SeemsByWordMatch : return nil error if Track song name, artist and featurings are contained in sequence
func (track Track) SeemsByWordMatch(sequence string) error {
	sequence = slug.Make(strings.ToLower(sequence))
	for _, trackItem := range append([]string{track.Song, track.Artist}, track.Featurings...) {
		trackItem = strings.ToLower(trackItem)
		if len(trackItem) > 7 && trackItem[:7] == "cast of" {
			trackItem = strings.Replace(trackItem, "cast of", "", -1)
		} else if len(trackItem) > 5 && trackItem[len(trackItem)-5:] == " cast" {
			trackItem = strings.Replace(trackItem, "cast", "", -1)
		}
		trackItem = strings.Replace(trackItem, " & ", " and ", -1)
		if strings.Contains(trackItem, " and ") {
			trackItem = strings.Split(trackItem, " and ")[0]
		}
		trackItem = strings.TrimSpace(trackItem)
		trackItem = slug.Make(trackItem)
		if len(trackItem) > 3 && !strings.Contains(sequence, trackItem) {
			return fmt.Errorf("Songs seem to be mismatching by words comparison: \"%v+\" in \"%s\", due to \"%s\"",
				append([]string{track.Song, track.Artist}, track.Featurings...), sequence, trackItem)
		}
	}
	return nil
}
