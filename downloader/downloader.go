package downloader

import (
	"errors"
	"os"
	"path/filepath"
)

var downloaders = []Downloader{}

type Downloader interface {
	supports(string) bool
	download(string, string, ...chan []byte) error
}

func Download(url, path string, channels ...chan []byte) error {
	if bytes, err := os.ReadFile(path); err == nil {
		for _, ch := range channels {
			ch <- bytes
		}
		return nil
	}

	for _, downloader := range downloaders {
		if downloader.supports(url) {
			if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
				return err
			}

			return downloader.download(url, path, channels...)
		}
	}
	return errors.New("unsupported url")
}
