package provider

import (
	"../track"
)

// Providers is an exported map of usable providers
var Providers = map[string]Provider{
	"YouTube": new(YouTubeProvider),
}

// Entry : single YouTube search result struct
type Entry struct {
	// TODO: what about dropping it?
	Track *track.Track

	ID            string
	URL           string
	Title         string
	User          string
	Duration      int
	AffinityScore int
}

// Provider defines the generic interface on which every download provider
// should be basing its logic
type Provider interface {
	Query(*track.Track) ([]*Entry, error)
	Match(*Entry, *track.Track) error
	Download(*Entry) error
	ValidateURL(url string) error
}
