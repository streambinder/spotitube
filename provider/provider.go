package provider

import (
	"context"
	"sort"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/entity"
)

var providers = []Provider{}

type Match struct {
	URL   string
	Score int
}

type Provider interface {
	Search(track *entity.Track) ([]*Match, error)
}

func Search(track *entity.Track) ([]*Match, error) {
	var (
		workers []nursery.ConcurrentJob
		matches []*Match
	)
	for _, provider := range providers {
		workers = append(workers, func(p Provider) func(ctx context.Context, ch chan error) {
			return func(ctx context.Context, ch chan error) {
				scopedMatches, err := p.Search(track)
				if err != nil {
					ch <- err
					return
				}
				matches = append(matches, scopedMatches...)
			}
		}(provider))
	}

	if err := nursery.RunConcurrently(workers...); err != nil {
		return nil, err
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches, nil
}
