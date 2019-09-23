package track

import (
	"os"
	"path/filepath"

	"github.com/streambinder/spotitube/src/system"
)

// TracksIndex : Tracks index keeping ID - filename mapping and eventual filename links
type TracksIndex struct {
	Tracks map[string]string
	Links  map[string][]string
}

var (
	ch = make(chan bool, 1)
)

// Index triggers a path scan searching for media files
// and populating a TracksIndex object in return
func Index(path string) *TracksIndex {
	i := TracksIndex{Tracks: make(map[string]string), Links: make(map[string][]string)}

	go func() {
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if info == nil || info.IsDir() {
				return nil
			}

			if linkPath, err := os.Readlink(path); err == nil {
				i.Links[linkPath] = append(i.Links[linkPath], path)
			} else if filepath.Ext(path) == ".mp3" {
				id := GetTag(path, ID3FrameSpotifyID)
				if len(id) > 0 {
					i.Tracks[id] = path
				}
			}

			return nil
		})

		ch <- true
	}()

	return &i
}

// IndexWait returns as Index(path) function is done
func IndexWait() {
	<-ch
}

// Sync flushes tracks index object on disk at input passed path
func (index *TracksIndex) Sync(path string) error {
	return system.DumpGob(path, index)
}
