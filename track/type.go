package track

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gosimple/slug"
)

const (
	typeAlbum = iota
	typeLive
	typeCover
	typeRemix
	typeAcoustic
	typeKaraoke
	typeParody
	typeReverse
)

var (
	types = []int{
		typeLive,
		typeCover,
		typeRemix,
		typeAcoustic,
		typeKaraoke,
		typeParody,
	}
)

// Type returns track variant
func (track Track) Type() int {
	for _, songType := range types {
		if IsType(track.Title, songType) {
			return songType
		}
	}
	return typeAlbum
}

// IsType returns True if given sequence matches with selected given songType variant
func IsType(sequence string, songType int) (match bool) {
	var regexes []string
	switch songType {
	case typeLive:
		regexes = []string{slug.Make("@"), slug.Make("live"), slug.Make("perform"), slug.Make("tour"), "[1-2]{1}[0-9]{3}"}
		break
	case typeCover:
		regexes = []string{slug.Make("cover"), slug.Make("vs"), slug.Make("amateur")}
		break
	case typeRemix:
		regexes = []string{slug.Make("remix"), slug.Make("radio-edit")}
		break
	case typeAcoustic:
		regexes = []string{slug.Make("acoustic")}
		break
	case typeKaraoke:
		regexes = []string{slug.Make("karaoke"), slug.Make("instrumental")}
		break
	case typeParody:
		regexes = []string{slug.Make("parody")}
		break
	case typeReverse:
		regexes = []string{slug.Make("reverse")}
		break
	}

	match, _ = regexp.MatchString(
		fmt.Sprintf(`(\A|-)(%s)(-|\z)`, strings.Join(regexes, "|")),
		slug.Make(sequence))
	return
}

// Seems returns an error if given sequence does not match
// with track
func (track Track) Seems(sequence string) error {
	if err := track.SeemsByWordMatch(sequence); err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(sequence), "full album") {
		return fmt.Errorf("Item seems to be pointing to an album, not to a song")
	}
	for _, songType := range types {
		if IsType(sequence, songType) && track.Type() != songType {
			return fmt.Errorf("Songs seem to be of different types")
		}
	}
	return nil
}

// SeemsByWordMatch returns and error if given sequence
// contains track song name, artist and featurings
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
