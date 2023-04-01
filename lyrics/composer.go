package lyrics

import (
	"context"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/entity"
)

var composers = []Composer{}

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
		workers = append(workers, func(ctx context.Context, ch chan error) {
			scopedLyrics, err := composer.Search(track)
			if err != nil {
				ch <- err
				return
			}

			if len(scopedLyrics) > 0 {
				result = scopedLyrics
			}
		})
	}

	if err := nursery.RunConcurrently(workers...); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}
