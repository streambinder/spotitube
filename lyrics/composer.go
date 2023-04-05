package lyrics

import (
	"context"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/entity"
)

var composers = []Composer{}

type Composer interface {
	Search(*entity.Track, ...context.Context) ([]byte, error)
}

// not found entries return no error
func Search(track *entity.Track) ([]byte, error) {
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
	return result, nil
}
