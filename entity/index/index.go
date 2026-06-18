package index

import (
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gosimple/slug"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/entity/id3"
)

const (
	Offline   = iota // previously synced
	Online           // needs to be synced
	Flush            // explicitly set to be re-synced
	Installed        // synced and successfully installed
)

type Index struct {
	ids   map[string]int // track Spotify IDs for canonical matches across renames
	paths map[string]int // final paths catch same-song collisions across upstream IDs
	lock  sync.RWMutex
}

func keyFromTrackID(track *entity.Track) string {
	return keyFromID(track.ID)
}

func keyFromTrackPath(track *entity.Track) string {
	return keyFromPath(track.Path().Final())
}

func keyFromID(id string) string {
	return id
}

func keyFromPath(path string) string {
	return slug.Make(filepath.Base(path))
}

func New() *Index {
	return &Index{
		ids:   make(map[string]int),
		paths: make(map[string]int),
		lock:  sync.RWMutex{},
	}
}

func (index *Index) Build(path string, init ...int) error {
	return index.BuildWithProgress(path, nil, init...)
}

func (index *Index) BuildWithProgress(path string, indexed chan<- string, init ...int) error {
	status := Offline
	for _, override := range init {
		status = override
	}

	return filepath.WalkDir(path, func(path string, entry fs.DirEntry, err error) error {
		// stop on root (or any subsequent inner directory, which is not relevant for us)
		// directory walk failure
		if err != nil {
			return err
		}

		// skip any inner directory from walk
		if entry.IsDir() && entry.Name() != filepath.Base(path) {
			return fs.SkipDir
		}

		// skip any file other than supported tracks
		if !strings.HasSuffix(filepath.Ext(path), entity.TrackFormat) {
			return nil
		}

		tag, err := id3.OpenSpotifyID(path)
		if err != nil {
			return err
		}

		if id := tag.SpotifyID(); len(id) > 0 {
			index.SetID(id, status)
			index.SetPath(path, status)
			if indexed != nil {
				indexed <- path
			}
		}

		return tag.Close()
	})
}

func (index *Index) Set(track *entity.Track, value int) {
	index.lock.Lock()
	defer index.lock.Unlock()
	if len(track.ID) > 0 {
		index.ids[keyFromTrackID(track)] = value
	}
	index.paths[keyFromTrackPath(track)] = value
}

func (index *Index) SetID(id string, value int) {
	index.lock.Lock()
	defer index.lock.Unlock()
	index.ids[keyFromID(id)] = value
}

func (index *Index) SetPath(path string, value int) {
	index.lock.Lock()
	defer index.lock.Unlock()
	index.paths[keyFromPath(path)] = value
}

func (index *Index) Get(track *entity.Track) (int, bool) {
	index.lock.RLock()
	defer index.lock.RUnlock()
	if len(track.ID) > 0 {
		value, ok := index.ids[keyFromTrackID(track)]
		if ok {
			return value, true
		}
	}

	value, ok := index.paths[keyFromTrackPath(track)]
	return value, ok
}

func (index *Index) Size(statuses ...int) (counter int) {
	index.lock.RLock()
	defer index.lock.RUnlock()

	if len(statuses) == 0 {
		return len(index.ids)
	}

	for _, value := range index.ids {
		for _, status := range statuses {
			if value == status {
				counter++
				break
			}
		}
	}
	return counter
}
