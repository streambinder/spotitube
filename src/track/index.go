package track

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/streambinder/spotitube/src/system"
)

// TracksIndex : Tracks index keeping ID - filename mapping and eventual filename links
type TracksIndex struct {
	Tracks map[string]string
}

var (
	ch = make(chan bool, 1)
)

// Index triggers a path scan searching for media files
// and populating a TracksIndex object in return
func Index(path string) *TracksIndex {
	i := TracksIndex{Tracks: make(map[string]string)}

	go func() {
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if info == nil || info.IsDir() {
				return nil
			}

			if filepath.Ext(path) == ".mp3" {
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

// Match returns whether an index element referenced by input id matches with input filename
func (index *TracksIndex) Match(id string, filename string) (string, bool, error) {
	if path, ok := index.Tracks[id]; ok {
		pathFilename := filepath.Base(path)
		return pathFilename, pathFilename == filepath.Base(filename), nil
	}

	return "", false, fmt.Errorf(fmt.Sprintf("Element with key %s does not exist", id))
}

// Rename replace input id element with input filename
func (index *TracksIndex) Rename(id string, filename string) {
	if path, ok := index.Tracks[id]; ok {
		index.Tracks[id] = path
	}
}
