package track

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bogem/id3v2"
	"github.com/streambinder/spotitube/system"
	"github.com/zmb3/spotify"
)

// Track : struct containing all the informations about a track
type Track struct {
	Title       string
	Song        string
	Artist      string
	Album       string
	Year        string
	Featurings  []string
	Genre       string
	TrackNumber int
	TrackTotals int
	Duration    int
	Image       string
	URL         string
	SpotifyID   string
	Lyrics      string
}

// TracksDump : Tracks dumpable object
type TracksDump struct {
	Tracks []*Track
	Time   time.Time
}

// CountOffline : return offline (local) songs count from Tracks
func CountOffline(tracks map[*Track]*SyncOptions) int {
	return len(tracks) - CountOnline(tracks)
}

// CountOnline : return online songs count from Tracks
func CountOnline(tracks map[*Track]*SyncOptions) int {
	var counter int
	for track := range tracks {
		if !track.Local() {
			counter++
		}
	}
	return counter
}

// OpenLocalTrack : parse local filename track informations into a new Track object
func OpenLocalTrack(filename string) (*Track, error) {
	if !system.FileExists(filename) {
		return new(Track), fmt.Errorf(fmt.Sprintf("%s does not exist", filename))
	}

	trackMp3, err := id3v2.Open(filename, id3v2.Options{Parse: true})
	if err != nil {
		return new(Track), fmt.Errorf(fmt.Sprintf("Cannot read id3 tags from \"%s\": %s", filename, err.Error()))
	}

	track := Track{
		Title:       TagGetFrame(trackMp3, ID3FrameTitle),
		Song:        TagGetFrame(trackMp3, ID3FrameSong),
		Artist:      TagGetFrame(trackMp3, ID3FrameArtist),
		Album:       TagGetFrame(trackMp3, ID3FrameAlbum),
		Year:        TagGetFrame(trackMp3, ID3FrameYear),
		Featurings:  strings.Split(TagGetFrame(trackMp3, ID3FrameFeaturings), "|"),
		Genre:       TagGetFrame(trackMp3, ID3FrameGenre),
		TrackNumber: 0,
		TrackTotals: 0,
		Duration:    0,
		Image:       TagGetFrame(trackMp3, ID3FrameArtworkURL),
		URL:         TagGetFrame(trackMp3, ID3FrameOrigin),
		SpotifyID:   TagGetFrame(trackMp3, ID3FrameSpotifyID),
		Lyrics:      TagGetFrame(trackMp3, ID3FrameLyrics),
	}

	if trackNumber, trackNumberErr := strconv.Atoi(TagGetFrame(trackMp3, ID3FrameTrackNumber)); trackNumberErr == nil {
		track.TrackNumber = trackNumber
	}
	if trackTotals, trackTotalsErr := strconv.Atoi(TagGetFrame(trackMp3, ID3FrameTrackTotals)); trackTotalsErr == nil {
		track.TrackTotals = trackTotals
	}
	if duration, durationErr := strconv.Atoi(TagGetFrame(trackMp3, ID3FrameDuration)); durationErr == nil {
		track.Duration = duration
	}

	trackMp3.Close()
	return &track, nil
}

// ParseSpotifyTrack : parse Spotify track into a new Track object
func ParseSpotifyTrack(spotifyTrack *spotify.FullTrack, spotifyAlbum *spotify.FullAlbum) *Track {
	track := Track{
		Title:  spotifyTrack.SimpleTrack.Name,
		Artist: (spotifyTrack.SimpleTrack.Artists[0]).Name,
		Album:  spotifyTrack.Album.Name,
		Year: func() string {
			if spotifyAlbum.ReleaseDatePrecision == "year" {
				return spotifyAlbum.ReleaseDate
			} else if strings.Contains(spotifyAlbum.ReleaseDate, "-") {
				return strings.Split(spotifyAlbum.ReleaseDate, "-")[0]
			}
			return "0000"
		}(),
		Featurings: func() []string {
			var featurings []string
			if len(spotifyTrack.SimpleTrack.Artists) > 1 {
				for _, artistItem := range spotifyTrack.SimpleTrack.Artists[1:] {
					featurings = append(featurings, artistItem.Name)
				}
			}
			return featurings
		}(),
		Genre: func() string {
			if len(spotifyAlbum.Genres) > 0 {
				return spotifyAlbum.Genres[0]
			}
			return ""
		}(),
		TrackNumber: spotifyTrack.SimpleTrack.TrackNumber,
		TrackTotals: len(spotifyAlbum.Tracks.Tracks),
		Duration:    spotifyTrack.SimpleTrack.Duration / 1000,
		Image: func() string {
			if len(spotifyTrack.Album.Images) > 0 {
				return spotifyTrack.Album.Images[0].URL
			}
			return ""
		}(),
		URL:       "",
		SpotifyID: spotifyTrack.SimpleTrack.ID.String(),
		Lyrics:    "",
	}

	track.Title, track.Song = parseTitle(track.Title, track.Featurings)

	track.Album = strings.Replace(track.Album, "[", "(", -1)
	track.Album = strings.Replace(track.Album, "]", ")", -1)
	track.Album = strings.Replace(track.Album, "{", "(", -1)
	track.Album = strings.Replace(track.Album, "}", ")", -1)

	if track.Local() {
		track.URL = track.GetID3Frame(ID3FrameOrigin)
		track.Lyrics = track.GetID3Frame(ID3FrameLyrics)
	}

	return &track
}

func parseTitle(trackTitle string, trackFeaturings []string) (string, string) {
	var trackSong string

	if !(strings.Contains(trackTitle, " (") && strings.Contains(strings.Split(strings.Split(trackTitle, ")")[0], "(")[1], " - ")) {
		trackTitle = strings.Split(trackTitle, " - ")[0]
	}
	if strings.Contains(trackTitle, " live ") {
		trackTitle = strings.Split(trackTitle, " live ")[0]
	}
	trackTitle = strings.TrimSpace(trackTitle)
	if len(trackFeaturings) > 0 {
		var (
			featuringsAlreadyParsed bool
			featuringSymbols        = []string{"featuring", "feat", "ft", "with", "prod"}
		)
		for _, featuringValue := range trackFeaturings {
			for _, featuringSymbol := range featuringSymbols {
				titleParts := strings.Split(strings.ToLower(trackTitle), featuringSymbol)
				if len(titleParts) > 1 && strings.Contains(titleParts[1], strings.ToLower(featuringValue)) {
					featuringsAlreadyParsed = true
				}
			}
		}
		if featuringsAlreadyParsed {
			for _, featuringSymbol := range featuringSymbols {
				for _, featuringSymbolCase := range []string{featuringSymbol, strings.Title(featuringSymbol)} {
					trackTitle = strings.Replace(trackTitle, featuringSymbolCase+". ", "ft. ", -1)
					trackTitle = strings.Replace(trackTitle, featuringSymbolCase+" ", "ft. ", -1)
				}
			}
		} else {
			if strings.Contains(trackTitle, "(") &&
				(strings.Contains(trackTitle, " vs. ") || strings.Contains(trackTitle, " vs ")) &&
				strings.Contains(trackTitle, ")") {
				trackTitle = strings.Split(trackTitle, " (")[0]
			}
			var trackFeaturingsInline string
			if len(trackFeaturings) > 1 {
				trackFeaturingsInline = "(ft. " + strings.TrimSpace(strings.Join(trackFeaturings[:len(trackFeaturings)-1], ", ")) +
					" and " + strings.TrimSpace(trackFeaturings[len(trackFeaturings)-1]) + ")"
			} else {
				trackFeaturingsInline = "(ft. " + strings.TrimSpace(trackFeaturings[0]) + ")"
			}
			trackTitle = fmt.Sprintf("%s %s", trackTitle, trackFeaturingsInline)
		}
		trackSong = strings.Split(trackTitle, " (ft. ")[0]
	} else {
		trackSong = trackTitle
	}

	return trackTitle, trackSong
}
