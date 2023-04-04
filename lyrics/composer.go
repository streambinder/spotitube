package lyrics

import (
	"context"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/entity"
)

var composers = []any{}

type Composer interface {
	Search(track *entity.Track) ([]byte, error)
}

// not found entries return no error
func Search(track *entity.Track) ([]byte, error) {
	var (
		workers []nursery.ConcurrentJob
		result  []byte
	)
	for _, composer := range composers {
		workers = append(workers, func(c Composer) func(ctx context.Context, ch chan error) {
			return func(ctx context.Context, ch chan error) {
				scopedLyrics, err := c.Search(track)
				if err != nil {
					ch <- err
					return
				}

				if len(scopedLyrics) > 0 {
					result = scopedLyrics
				}
			}
		}(composer.(Composer)))
	}

	if err := nursery.RunConcurrently(workers...); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}
