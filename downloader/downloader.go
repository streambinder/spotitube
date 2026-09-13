package downloader

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/streambinder/spotitube/processor"
)

var downloaders = []Downloader{}

type Downloader interface {
	supports(context.Context, string) bool
	download(context.Context, string, string, processor.Processor, ...chan []byte) error
}

func Download(ctx context.Context, url, path string, processor processor.Processor, channels ...chan []byte) error {
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
		if downloader.supports(ctx, url) {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}

			return downloader.download(ctx, url, path, processor, channels...)
		}
	}
	return errors.New("unsupported url: " + url)
}
