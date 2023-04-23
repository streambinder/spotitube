package index

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/entity/id3"
)

const (
	Offline   = iota // previously synced
	Online           // needs to be synced
	Installed        // synced and successfully installed
)

type Index map[string]int

func New() Index {
	return make(map[string]int)
}

func (index Index) Build(path string, init ...int) error {
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
		if entry.IsDir() {
			return fs.SkipDir
		}

		// skip any file other than supported tracks
		if !strings.HasSuffix(filepath.Ext(path), entity.TrackFormat) {
			return nil
		}

		tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
		if err != nil {
			return err
		}

		if frame, ok := tag.GetLastFrame(id3.FrameSpotifyID).(id3v2.UnknownFrame); !ok {
			return errors.New("cannot parse Spotify ID frame")
		} else {
			index[string(frame.Body)] = status
		}

		return tag.Close()
	})
}
