package youtube

import (
	spttb_track "track"

	"github.com/PuerkitoBio/goquery"
)

// Tracks : simple iterator-like struct to easily loop over YouTube search results
type Tracks struct {
	Track             *spttb_track.Track
	Selection         *goquery.Selection
	SelectionDesc     *goquery.Selection
	SelectionDuration *goquery.Selection
	SelectionPointer  int
}

// Track : single YouTube search result struct
type Track struct {
	Track    *spttb_track.Track
	ID       string
	URL      string
	Title    string
	User     string
	Duration int
}
