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
	Title        string
	Artist       string
	Album        string
	Filename     string
	FilenameTemp string
	FilenameExt  string
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
	track.Filename = track.Artist + " - " + track.Title
	track.Filename = strings.Replace(track.Filename, "/", "", -1)
	track.Filename = strings.Replace(track.Filename, "  ", " ", -1)
	track.Filename = strings.Replace(track.Filename, ".", "", -1)
	track.Filename = sanitize.Accents(track.Filename)
	track.Filename = strings.TrimSpace(track.Filename)
	track.FilenameTemp = sanitize.Name("." + track.Filename)
	return track
}
