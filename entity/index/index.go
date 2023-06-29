package index

import (
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bogem/id3v2/v2"
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
	data map[string]int
	lock sync.RWMutex
}

func keyFromTrack(track *entity.Track) string {
	return keyFromPath(track.Path().Final())
}

func keyFromPath(path string) string {
	return slug.Make(filepath.Base(path))
}

func New() *Index {
	return &Index{
		make(map[string]int),
		sync.RWMutex{},
	}
}

func (index *Index) Build(path string, init ...int) error {
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
		if entry.IsDir() && entry.Name() != path {
			return fs.SkipDir
		}

		// skip any file other than supported tracks
		if !strings.HasSuffix(filepath.Ext(path), entity.TrackFormat) {
			return nil
		}

		tag, err := id3.Open(path, id3v2.Options{Parse: true})
		if err != nil {
			return err
		}

		if id := tag.SpotifyID(); len(id) > 0 {
			index.SetPath(path, status)
		}

		return tag.Close()
	})
}

func (index *Index) Set(track *entity.Track, value int) {
	index.lock.Lock()
	defer index.lock.Unlock()
	index.data[keyFromTrack(track)] = value
}

func (index *Index) SetPath(path string, value int) {
	index.lock.Lock()
	defer index.lock.Unlock()
	index.data[keyFromPath(path)] = value
}

func (index *Index) Get(track *entity.Track) (int, bool) {
	index.lock.RLock()
	defer index.lock.RUnlock()
	value, ok := index.data[keyFromTrack(track)]
	return value, ok
}

func (index *Index) Size(statuses ...int) (counter int) {
	if len(statuses) == 0 {
		return len(index.data)
	}

	for _, value := range index.data {
		for _, status := range statuses {
			if value == status {
				counter += 1
				break
			}
		}
	}
	return
}
