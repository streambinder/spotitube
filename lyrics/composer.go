package lyrics

import (
	"context"
	"os"
	"path/filepath"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/entity"
)

var composers = []Composer{}

type Composer interface {
	Search(*entity.Track, ...context.Context) ([]byte, error)
}

// not found entries return no error
func Search(track *entity.Track) ([]byte, error) {
	if bytes, err := os.ReadFile(track.Path().Lyrics()); err == nil {
		return bytes, nil
	}

	var (
		workers        []nursery.ConcurrentJob
		result         []byte
		ctxBackground  = context.Background()
		ctx, ctxCancel = context.WithCancel(ctxBackground)
	)
	defer ctxCancel()

	for _, composer := range composers {
		workers = append(workers, func(c Composer) func(context.Context, chan error) {
			return func(ctx context.Context, ch chan error) {
				scopedLyrics, err := c.Search(track, ctx)
				if err != nil {
					ch <- err
					return
				}

				if len(scopedLyrics) > len(result) {
					result = scopedLyrics
					ctxCancel()
				}
			}
		}(composer))
	}

	if err := nursery.RunConcurrentlyWithContext(ctx, workers...); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	if err := os.MkdirAll(filepath.Dir(track.Path().Lyrics()), os.ModePerm); err != nil {
		return nil, err
	}

	return result, os.WriteFile(track.Path().Lyrics(), result, 0o644)
}
