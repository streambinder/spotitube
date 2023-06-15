package downloader

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/streambinder/spotitube/processor"
)

var downloaders = []Downloader{}

type Downloader interface {
	supports(string) bool
	download(string, string, processor.Processor, ...chan []byte) error
}

func Download(url, path string, processor processor.Processor, channels ...chan []byte) error {
	if len(url) == 0 {
		return nil
	}

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

			return downloader.download(url, path, processor, channels...)
		}
	}
	return errors.New("unsupported url: " + url)
}
