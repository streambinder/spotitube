package provider

import (
	"fmt"
	"math"
	"strings"

	"github.com/agnivade/levenshtein"
	"github.com/gosimple/slug"
	"github.com/streambinder/spotitube/src/track"
)

const (
	// DurationTolerance : max result duration difference tolerance
	DurationTolerance = 20 // second(s)
)

// All return the array of usable providers
func All() []Provider {
	return []Provider{
		new(YouTubeProvider),
	}
}

// For returns a provider for a given URL
func For(URL string) (Provider, error) {
	for _, p := range All() {
		if err := p.ValidateURL(URL); err == nil {
			return p, nil
		}
	}

	return nil, fmt.Errorf("No provider found")
}

// Entry : single search result struct
type Entry struct {
	ID       string
	URL      string
	Title    string
	User     string
	Duration int
}

// Empty returns true if entry does not have a URL and it's unusable, then
func (e *Entry) Empty() bool {
	return e.URL == ""
}

// Repr returns a human readable representation of the entry
func (e *Entry) Repr() string {
	if e.Title != "" {
		return e.Title
	} else if !e.Empty() {
		return e.URL
	}

	return ""
}

// Provider defines the generic interface on which every download provider
// should be basing its logic
type Provider interface {
	Name() string
	Query(*track.Track) ([]*Entry, error)
	Match(*Entry, *track.Track) error
	Download(*Entry, string) error
	ValidateURL(url string) error
}

// Scorable defines the functions needed to apply a score over results
type Scorable interface {
	Score(*Entry, *track.Track) int
}

// Scorer provides a basic Scorable implementation
type Scorer struct {
	Scorable
}

// Score implements a basic scoring logic usable by any Provider
func (s Scorer) Score(e *Entry, t *track.Track) int {
	var score = 0 - levenshtein.ComputeDistance(t.Query(), fmt.Sprintf("%s %s", e.User, e.Title))

	if math.Abs(float64(t.Duration-e.Duration)) <= float64(DurationTolerance/2) {
		score += 20
	} else if math.Abs(float64(t.Duration-e.Duration)) <= float64(DurationTolerance) {
		score += 10
	}

	if err := t.SeemsByWordMatch(fmt.Sprintf("%s %s", e.User, e.Title)); err == nil {
		score += 10
	}

	if strings.Contains(slug.Make(e.User), slug.Make(t.Artist)) {
		score += 10
	}

	if track.IsType(e.Title, t.Type()) {
		score += 10
	}

	return score
}
