package youtube

import (
	spttb_track "track"
)

// Tracks : Track array
type Tracks []Track

// Track : single YouTube search result struct
type Track struct {
	Track         *spttb_track.Track
	ID            string
	URL           string
	Title         string
	User          string
	Duration      int
	AffinityScore int
}
