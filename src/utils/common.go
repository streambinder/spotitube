package utils

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/kennygrant/sanitize"
	spotify "github.com/zmb3/spotify"
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

func MakeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

func GetBoolPointer(value bool) *bool {
	return &value
}

// spotify-dl utils

const (
	LogNormal  = iota
	LogDebug   = iota
	LogWarning = iota
	LogFatal   = iota
)

var (
	enable_logfile *bool = GetBoolPointer(false)
	enable_debug   *bool = GetBoolPointer(false)
)

type Logger struct {
	Prefix string
	Color  func(a ...interface{}) string
	File   string
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
		File:   DEFAULT_LOG_PATH,
	}
	return logger
}
func (logger Logger) UncoloredPrefix() string {
	space_pre := strings.Repeat(" ", ((SHELL_NAME_MIN_LENGTH - len(logger.Prefix)) / 2))
	space_post := space_pre
	if len(logger.Prefix)%2 == 1 {
		space_post = space_post + " "
	}
	return "[" + space_pre + strings.ToUpper(logger.Prefix) + space_post + "]"
}

func (logger Logger) ColoredPrefix() string {
	return logger.Color(logger.UncoloredPrefix())
}

func (logger Logger) LogOpt(message string, level int) {
	if !(*enable_debug) && level == LogDebug {
		return
	}
	if *enable_logfile {
		logger.LogWrite(message)
	}
	if level == LogDebug {
		message = color.MagentaString(message)
	} else if level == LogWarning {
		message = color.YellowString(message)
	} else if level == LogFatal {
		message = color.RedString(message)
	}
	fmt.Println(logger.ColoredPrefix(), message)
	if level == LogFatal {
		os.Exit(1)
	}
}

func (logger Logger) Log(message string) {
	logger.LogOpt(message, LogNormal)
}

func (logger Logger) Debug(message string) {
	logger.LogOpt(message, LogDebug)
}

func (logger Logger) Warn(message string) {
	logger.LogOpt(message, LogWarning)
}

func (logger Logger) Fatal(message string) {
	logger.LogOpt(message, LogFatal)
}

func (logger Logger) LogWrite(message string) {
	logfile, err := os.OpenFile(logger.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer logfile.Close()
	if _, err = logfile.WriteString(time.Now().Format("2006-01-02 15:04:05") + " " +
		logger.UncoloredPrefix() + " " +
		message + "\n"); err != nil {
		panic(err)
	}
}

func (logger Logger) SetFile(path string) {
	logger.EnableLogFile()
	logger.File = path
}

func (logger Logger) EnableLogFile() {
	enable_logfile = GetBoolPointer(true)
}

func (logger Logger) EnableDebug() {
	enable_debug = GetBoolPointer(true)
}

const (
	SongTypeAlbum    = iota
	SongTypeLive     = iota
	SongTypeCover    = iota
	SongTypeRemix    = iota
	SongTypeAcoustic = iota
	SongTypeKaraoke  = iota
)

type Track struct {
	Title         string
	Song          string
	Artist        string
	Album         string
	Featurings    []string
	Image         spotify.Image
	Filename      string
	FilenameTemp  string
	FilenameExt   string
	SearchPattern string
	SongType      int
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
	track.SongType = SongTypeAlbum
	for song_type := range []int{SongTypeLive, SongTypeCover, SongTypeRemix, SongTypeAcoustic, SongTypeKaraoke} {
		if SeemsType(track.Title, song_type) {
			track.SongType = song_type
		}
	}

	track.Title = strings.Split(track.Title, " - ")[0]
	if strings.Contains(track.Title, " live ") {
		track.Title = strings.Split(track.Title, " live ")[0]
	}
	track.Title = strings.TrimSpace(track.Title)
	if len(track.Featurings) > 0 {
		if strings.Contains(strings.ToLower(track.Title), "feat. ") || strings.Contains(strings.ToLower(track.Title), "ft. ") {
			track.Title = strings.Replace(track.Title, "feat. ", "ft. ", -1)
		} else {
			var track_featurings string
			if len(track.Featurings) > 1 {
				track_featurings = "(ft. " + strings.Join(track.Featurings[:len(track.Featurings)-1], ", ") +
					" and " + track.Featurings[len(track.Featurings)-1] + ")"
			} else {
				track_featurings = "(ft. " + track.Featurings[0] + ")"
			}
			track.Title = track.Title + " " + track_featurings
		}
		track.Song = strings.Split(track.Title, " (ft. ")[0]
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
	if !track.SeemsByWordMatch(sequence) ||
		strings.Contains(strings.ToLower(sequence), "full album") {
		return false
	}

	for song_type := range []int{SongTypeLive, SongTypeCover, SongTypeRemix, SongTypeAcoustic, SongTypeKaraoke} {
		if SeemsType(sequence, song_type) && track.SongType != song_type {
			return false
		}
	}

	return true
}

func (track Track) SeemsByWordMatch(sequence string) bool {
	sequence = sanitize.Name(strings.ToLower(sequence))
	for _, track_item := range append([]string{track.Song, track.Artist}, track.Featurings...) {
		track_item = strings.ToLower(track_item)
		if len(track_item) > 7 && track_item[:7] == "cast of" {
			track_item = strings.Replace(track_item, "cast of", "", -1)
		} else if len(track_item) > 5 && track_item[len(track_item)-5:] == " cast" {
			track_item = strings.Replace(track_item, "cast", "", -1)
		}
		track_item = strings.Replace(track_item, " & ", " and ", -1)
		if strings.Contains(track_item, " and ") {
			track_item = strings.Split(track_item, " and ")[0]
		}
		track_item = strings.TrimSpace(track_item)
		track_item = sanitize.Name(track_item)
		if !strings.Contains(sequence, track_item) {
			return false
		}
	}
	return true
}

func SeemsType(sequence string, song_type int) bool {
	var song_type_aliases []string
	if song_type == SongTypeLive {
		song_type_aliases = []string{"@", "live", "perform", "tour"}
		for _, year := range MakeRange(1950, 2050) {
			song_type_aliases = append(song_type_aliases, strconv.Itoa(year))
		}
	} else if song_type == SongTypeCover {
		song_type_aliases = []string{"cover", "vs"}
	} else if song_type == SongTypeRemix {
		song_type_aliases = []string{"remix", "radio edit"}
	} else if song_type == SongTypeAcoustic {
		song_type_aliases = []string{"acoustic"}
	} else if song_type == SongTypeKaraoke {
		song_type_aliases = []string{"karaoke"}
	}

	for _, song_type_alias := range song_type_aliases {
		sequence_tmp := sequence
		if len(song_type_alias) == 1 {
			sequence_tmp = strings.ToLower(sequence)
		} else {
			sequence_tmp = sanitize.Name(strings.ToLower(sequence))
		}
		if len(sanitize.Name(strings.ToLower(song_type_alias))) == len(song_type_alias) {
			song_type_alias = sanitize.Name(strings.ToLower(song_type_alias))
		}
		if strings.Contains(sequence_tmp, song_type_alias) {
			return true
		}
	}
	return false
}
