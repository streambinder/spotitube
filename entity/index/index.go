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

	return filepath.WalkDir(path, func(walkPath string, entry fs.DirEntry, err error) error {
		// a single unreadable entry must not abort the whole indexing:
		// skip it and keep scanning the rest of the library. A failure
		// on the library root itself stays fatal, otherwise an empty
		// index would silently mark every track as new
		if err != nil {
			if walkPath == path {
				return err
			}
			return nil
		}

		// skip any inner directory from walk
		if entry.IsDir() && walkPath != path {
			return fs.SkipDir
		}

		// skip any file other than supported tracks
		if !strings.EqualFold(filepath.Ext(walkPath), "."+entity.TrackFormat) {
			return nil
		}

		// a single corrupt or unreadable file must not abort the whole indexing
		tag, err := id3.OpenSpotifyID(walkPath)
		if err != nil {
			return nil
		}

		if id := tag.SpotifyID(); len(id) > 0 {
			index.SetID(id, status)
			index.SetPath(walkPath, status)
			if indexed != nil {
				indexed <- walkPath
			}
		}

		// a close failure must not abort the whole indexing either
		_ = tag.Close()
		return nil
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

func (index *Index) GetID(id string) (int, bool) {
	index.lock.RLock()
	defer index.lock.RUnlock()
	value, ok := index.ids[keyFromID(id)]
	return value, ok
}

func (index *Index) GetPath(path string) (int, bool) {
	index.lock.RLock()
	defer index.lock.RUnlock()
	value, ok := index.paths[keyFromPath(path)]
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
