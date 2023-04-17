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
	search(*entity.Track, ...context.Context) ([]byte, error)
}

// not found entries return no error
func Search(track *entity.Track) (string, error) {
	if bytes, err := os.ReadFile(track.Path().Lyrics()); err == nil {
		return string(bytes), nil
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
				scopedLyrics, err := c.search(track, ctx)
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
		return "", err
	}

	if len(result) == 0 {
		return "", nil
	}

	if err := os.MkdirAll(filepath.Dir(track.Path().Lyrics()), os.ModePerm); err != nil {
		return "", err
	}

	return string(result), os.WriteFile(track.Path().Lyrics(), result, 0o644)
}
