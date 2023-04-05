package downloader

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/streambinder/spotitube/util"
)

var downloaders = []Downloader{}

type Downloader interface {
	Supports(url string) bool
	Download(url, path string) error
}

func Download(url, path string) error {
	if err := util.ErrOnly(os.Stat(path)); err == nil {
		return nil
	}

	for _, downloader := range downloaders {
		if downloader.Supports(url) {
			if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
				return err
			}

			return downloader.Download(url, path)
		}
	}
	return errors.New("unsupported url")
}
