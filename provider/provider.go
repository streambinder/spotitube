package provider

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/entity"
)

var (
	providers  = []Provider{}
	misleading = []string{"cover", "live", "karaoke", "performance", "studio", "instrumental", "remix", "acoustic"}
)

type Match struct {
	URL   string
	Score int
}

type Provider interface {
	search(track *entity.Track) ([]*Match, error)
}

func Search(track *entity.Track) ([]*Match, error) {
	var (
		workers  []nursery.ConcurrentJob
		matches  []*Match
		mu       sync.Mutex
		errCount int
	)
	for _, provider := range providers {
		workers = append(workers, func(p Provider) func(ctx context.Context, ch chan error) {
			return func(_ context.Context, _ chan error) {
				scopedMatches, err := p.search(track)
				if err != nil {
					mu.Lock()
					errCount++
					mu.Unlock()
					return
				}
				mu.Lock()
				matches = append(matches, scopedMatches...)
				mu.Unlock()
			}
		}(provider))
	}

	if err := nursery.RunConcurrently(workers...); err != nil {
		return nil, err
	}

	if errCount == len(workers) {
		return nil, errors.New("all providers failed")
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches, nil
}
