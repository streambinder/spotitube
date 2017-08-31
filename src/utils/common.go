package utils

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/kennygrant/sanitize"
)

// system utils

func IsDir(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	file_stat, err := file.Stat()
	if err != nil {
		return false
	}
	return file_stat.IsDir()
}

// spotify-dl utils

type Logger struct {
	Prefix string
	Color  func(a ...interface{}) string
}

func NewLogger() Logger {
	var shell_color func(a ...interface{}) string = color.New(SHELL_COLOR_DEFAULT).SprintFunc()
	var caller_package string = SHELL_NAME_DEFAULT

	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if ok && details != nil && details.Name()[:4] != "main" {
		caller_package = strings.Split(details.Name(), ".")[0]
		if caller_package == "spotify" {
			shell_color = color.New(SHELL_COLOR_SPOTIFY).SprintFunc()
		} else if caller_package == "youtube" {
			shell_color = color.New(SHELL_COLOR_YOUTUBE).SprintFunc()
		}
	}
	logger := Logger{
		Prefix: caller_package,
		Color:  shell_color,
	}
	return logger
}

func (logger Logger) ColoredPrefix() string {
	space_pre := strings.Repeat(" ", ((SHELL_NAME_MIN_LENGTH - len(logger.Prefix)) / 2))
	space_post := space_pre
	if len(logger.Prefix)%2 == 1 {
		space_post = space_post + " "
	}
	return logger.Color("[" + space_pre + strings.ToUpper(logger.Prefix) + space_post + "]")
}

func (logger Logger) Log(message string, fatal ...bool) {
	var is_fatal bool = (len(fatal) == 1 && fatal[0])
	if is_fatal {
		message = color.RedString(message)
	}
	fmt.Println(logger.ColoredPrefix(), message)
	if is_fatal {
		os.Exit(1)
	}
}

func (logger Logger) Fatal(message string) {
	logger.Log(message, true)
}

type Track struct {
	Title         string
	Song          string
	Artist        string
	Album         string
	Featurings    []string
	Filename      string
	FilenameTemp  string
	FilenameExt   string
	SearchPattern string
}

type Tracks []Track

const (
	SongTypeLive     = iota
	SongTypeCover    = iota
	SongTypeRemix    = iota
	SongTypeAcoustic = iota
)

func (tracks Tracks) Has(track Track) bool {
	for _, having_track := range tracks {
		if strings.ToLower(having_track.Filename) == strings.ToLower(track.Filename) {
			return true
		}
	}
	return false
}

func (track Track) Normalize() Track {
	track.Title = strings.Split(track.Title, " - ")[0]
	if strings.Contains(track.Title, " live ") {
		track.Title = strings.Split(track.Title, " live ")[0]
	}
	track.Title = strings.TrimSpace(track.Title)
	if len(track.Featurings) > 0 {
		if strings.Contains(strings.ToLower(track.Title), "feat. ") || strings.Contains(strings.ToLower(track.Title), "ft. ") {
			track.Title = strings.Replace(track.Title, "ft. ", "feat. ", -1)
		} else {
			var track_featurings = "(feat. " + strings.Join(track.Featurings[:len(track.Featurings)-1], ", ") +
				" and " + track.Featurings[len(track.Featurings)-1] + ")"
			track.Title = track.Title + " " + track_featurings
		}
		track.Song = strings.Split(track.Title, " (feat. ")[0]
	} else {
		track.Song = track.Title
	}

	track.Album = strings.Replace(track.Album, "[", "(", -1)
	track.Album = strings.Replace(track.Album, "]", ")", -1)
	track.Album = strings.Replace(track.Album, "{", "(", -1)
	track.Album = strings.Replace(track.Album, "}", ")", -1)

	track.Filename = track.Artist + " - " + track.Title
	for _, symbol := range []string{"/", "\\", ".", "?", "<", ">", ":", "*"} {
		track.Filename = strings.Replace(track.Filename, symbol, "", -1)
	}
	track.Filename = strings.Replace(track.Filename, "  ", " ", -1)
	track.Filename = sanitize.Accents(track.Filename)
	track.Filename = strings.TrimSpace(track.Filename)
	track.FilenameTemp = sanitize.Name("." + track.Filename)

	track.SearchPattern = strings.Replace(track.FilenameTemp[1:], "-", " ", -1)

	return track
}

func (track Track) Seems(sequence string) bool {
	sequence_sanitized := sanitize.Name(strings.ToLower(sequence))

	for _, track_item := range append([]string{track.Song, track.Artist}, track.Featurings...) {
		track_item = strings.ToLower(track_item)
		if track_item[:7] == "cast of" {
			track_item = strings.Replace(track_item, "cast of")
		} else if track_item[len(track_item)-5:len(track_item)] == " cast" {
			track_item = strings.Replace(track_item, "cast")
		}
		track_item = strings.Replace(track_item, " & ", " and ", -1)
		if strings.Contains(track_item, " and ") {
			track_item = strings.Split(track_item, " and ")[0]
		}
		track_item = strings.TrimSpace(track_item)
		track_item = sanitize.Name(track_item)
		if !strings.Contains(sequence_sanitized, track_item) {
			return false
		}
	}

	if !SeemsType(track.Title, SongTypeLive) && SeemsType(sequence_sanitized, SongTypeLive) {
		return false
	} else if !SeemsType(track.Title, SongTypeCover) && SeemsType(sequence_sanitized, SongTypeCover) {
		return false
	} else if !SeemsType(track.Title, SongTypeRemix) && SeemsType(sequence_sanitized, SongTypeRemix) {
		return false
	} else if !SeemsType(track.Title, SongTypeAcoustic) && SeemsType(sequence_sanitized, SongTypeAcoustic) {
		return false
	}

	return true
}

func SeemsType(sequence string, song_type int) bool {
	sequence = strings.ToLower(sequence)
	matching := func(sequence string, song_type_alias string) bool {
		for _, symbol := range []string{"", "(", "[", "{"} {
			if strings.Contains(strings.ToLower(sequence), symbol+song_type_alias) {
				return true
			}
		}
		return false
	}

	var song_type_aliases []string
	if song_type == SongTypeLive {
		song_type_aliases = []string{"@", "live at", "perform", "tour"}
	} else if song_type == SongTypeCover {
		song_type_aliases = []string{"cover", "vs"}
	} else if song_type == SongTypeRemix {
		song_type_aliases = []string{"remix", "radio edit"}
	} else if song_type == SongTypeAcoustic {
		song_type_aliases = []string{"acoustic"}
	}

	for _, song_type_alias := range song_type_aliases {
		if matching(sequence, song_type_alias) {
			return true
		}
	}
	return false
}
