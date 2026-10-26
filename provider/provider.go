package provider

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/pango/entity"
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
	name() string
	search(track *entity.Track) ([]*Match, error)
}

// Search queries every provider concurrently and returns the matches
// ranked by score. A provider failure never discards the matches of
// the other providers: the returned error aggregates every failure,
// each prefixed with the provider name, alongside the matches. The
// callers surface that error, so a broken provider is not mistaken
// for a track that was not found. The error is nil only when every
// provider answered.
func Search(track *entity.Track) ([]*Match, error) {
	var (
		workers []nursery.ConcurrentJob
		matches []*Match
		errs    []error
		mu      sync.Mutex
	)
	for _, provider := range providers {
		workers = append(workers, func(p Provider) func(ctx context.Context, ch chan error) {
			return func(_ context.Context, _ chan error) {
				scopedMatches, err := p.search(track)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					errs = append(errs, fmt.Errorf("%s: %w", p.name(), err))
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

	if len(errs) > 0 {
		msgs := make([]string, 0, len(errs))
		for _, err := range errs {
			msgs = append(msgs, err.Error())
		}
		return matches, errors.New(strings.Join(msgs, "; "))
	}

	return matches, nil
}
