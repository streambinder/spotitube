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
	Artist        string
	Album         string
	Featurings    []string
	Filename      string
	FilenameTemp  string
	FilenameExt   string
	SearchPattern string
}

type Tracks []Track

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
	track.Title = track.Title + " (ft. " + strings.Join(track.Featurings, ", ") + ")"

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
	track_title := strings.ToLower(track.Title)
	track_title = strings.Replace(track_title, " & ", " and ", -1)
	for _, splitter := range []string{" and ", "feat. "} {
		if strings.Contains(track_title, splitter) {
			track_title = strings.Split(track_title, splitter)[0]
		}
	}
	track_title = sanitize.Name(track_title)

	track_artist := strings.ToLower(track.Artist)
	track_artist = strings.Replace(track_artist, " & ", " and ", -1)
	for _, splitter := range []string{" and "} {
		if strings.Contains(track_artist, splitter) {
			track_artist = strings.Split(track_artist, splitter)[0]
		}
	}
	track_artist = sanitize.Name(track_artist)

	b_live := strings.Contains(strings.ToLower(track.Title), " live at ")
	b_cover := strings.Contains(strings.ToLower(track.Title), " cover")
	b_remix := strings.Contains(strings.ToLower(track.Title), " remix")
	b_radioedit := strings.Contains(strings.ToLower(track.Title), " radio edit")
	b_acoustic := strings.Contains(strings.ToLower(track.Title), "acoustic")

	if strings.Contains(sequence_sanitized, track_title) && strings.Contains(sequence_sanitized, track_artist) {
		if !b_live && (strings.Contains(strings.ToLower(sequence), " live at ") ||
			strings.Contains(strings.ToLower(sequence), " @ ") ||
			strings.Contains(strings.ToLower(sequence), "(live")) {
			return false
		} else if !b_cover && (strings.Contains(strings.ToLower(sequence), " cover") ||
			strings.Contains(strings.ToLower(sequence), "(cover") ||
			strings.Contains(strings.ToLower(sequence), "[cover") ||
			strings.Contains(strings.ToLower(sequence), "{cover")) {
			return false
		} else if !b_remix && strings.Contains(strings.ToLower(sequence), " remix") {
			return false
		} else if !b_radioedit && strings.Contains(strings.ToLower(sequence), " radio edit") {
			return false
		} else if !b_acoustic && strings.Contains(strings.ToLower(sequence), "acoustic") {
			return false
		} else {
			return true
		}
	}
	return false
}
