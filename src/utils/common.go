package utils

import (
	"os"
	"strings"
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

type Track struct {
	Title  string
	Artist string
	Album  string
}

type Tracks []Track

func (tracks Tracks) Has(track Track) bool {
	track_title := strings.TrimSpace(strings.ToLower(track.Title))
	track_artist := strings.TrimSpace(strings.ToLower(track.Artist))
	for _, track := range tracks {
		if track_title == strings.TrimSpace(strings.ToLower(track.Title)) && track_artist == strings.TrimSpace(strings.ToLower(track.Artist)) {
			return true
		}
	}
	return false
}

func (track Track) Normalize() Track {
	track.Title = strings.Split(track.Title, " - ")[0]
	return track
}
