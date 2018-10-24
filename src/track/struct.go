package track

import (
	"time"
)

// Track : struct containing all the informations about a track
type Track struct {
	Title         string
	Song          string
	Artist        string
	Album         string
	Year          string
	Featurings    []string
	Genre         string
	TrackNumber   int
	TrackTotals   int
	Duration      int
	SongType      int
	Image         string
	URL           string
	Filename      string
	FilenameTemp  string
	FilenameExt   string
	SearchPattern string
	Lyrics        string
	Local         bool
}

// Tracks : Track array
type Tracks []Track

// TracksDump : Tracks dumpable object
type TracksDump struct {
	Tracks Tracks
	Time   time.Time
}
