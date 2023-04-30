package index

import (
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
		if entry.IsDir() && entry.Name() != path {
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

		for _, frame := range tag.GetFrames(tag.CommonID("User defined text information frame")) {
			frame, ok := frame.(id3v2.UserDefinedTextFrame)
			if !(ok && strings.EqualFold(frame.UniqueIdentifier(), id3.FrameSpotifyID)) {
				continue
			}

			index[frame.Value] = status
			break
		}

		return tag.Close()
	})
}
