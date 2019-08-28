package track

import (
	"github.com/bogem/id3v2"
)

const (
	// ID3FrameTitle : ID3 title frame tag identifier
	ID3FrameTitle = iota
	// ID3FrameSong : ID3 song frame tag identifier
	ID3FrameSong
	// ID3FrameArtist : ID3 artist frame tag identifier
	ID3FrameArtist
	// ID3FrameAlbum : ID3 album frame tag identifier
	ID3FrameAlbum
	// ID3FrameGenre : ID3 genre frame tag identifier
	ID3FrameGenre
	// ID3FrameYear : ID3 year frame tag identifier
	ID3FrameYear
	// ID3FrameFeaturings : ID3 featurings frame tag identifier
	ID3FrameFeaturings
	// ID3FrameTrackNumber : ID3 track number frame tag identifier
	ID3FrameTrackNumber
	// ID3FrameTrackTotals : ID3 total tracks number frame tag identifier
	ID3FrameTrackTotals
	// ID3FrameArtwork : ID3 artwork frame tag identifier
	ID3FrameArtwork
	// ID3FrameArtworkURL : ID3 artwork URL frame tag identifier
	ID3FrameArtworkURL
	// ID3FrameLyrics : ID3 lyrics frame tag identifier
	ID3FrameLyrics
	// ID3FrameOrigin : ID3 origin frame tag identifier
	ID3FrameOrigin
	// ID3FrameDuration : ID3 duration frame tag identifier
	ID3FrameDuration
	// ID3FrameSpotifyID : ID3 Spotify ID frame tag identifier
	ID3FrameSpotifyID
)

// GetTag : open, parse and return filename ID3 tag
func GetTag(path string, frame int) string {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return ""
	}
	defer tag.Close()

	return TagGetFrame(tag, frame)
}

// TagGetFrame : get input frame from open input Tag
func TagGetFrame(tag *id3v2.Tag, frame int) string {
	switch frame {
	case ID3FrameTitle:
		return tag.Title()
	case ID3FrameSong:
		return TagGetFrameSong(tag)
	case ID3FrameArtist:
		return tag.Artist()
	case ID3FrameAlbum:
		return tag.Album()
	case ID3FrameGenre:
		return tag.Genre()
	case ID3FrameYear:
		return tag.Year()
	case ID3FrameFeaturings:
		return TagGetFrameFeaturings(tag)
	case ID3FrameTrackNumber:
		return TagGetFrameTrackNumber(tag)
	case ID3FrameTrackTotals:
		return TagGetFrameTrackTotals(tag)
	case ID3FrameArtwork:
		return TagGetFrameArtwork(tag)
	case ID3FrameArtworkURL:
		return TagGetFrameArtworkURL(tag)
	case ID3FrameLyrics:
		return TagGetFrameLyrics(tag)
	case ID3FrameOrigin:
		return TagGetFrameOrigin(tag)
	case ID3FrameDuration:
		return TagGetFrameDuration(tag)
	case ID3FrameSpotifyID:
		return TagGetFrameSpotifyID(tag)
	}
	return ""
}

// TagGetFrameSong : get track song title frame from input Tag
func TagGetFrameSong(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "song" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagGetFrameFeaturings : get track featurings frame from input Tag
func TagGetFrameFeaturings(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "featurings" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagGetFrameTrackNumber : get track number frame from input Tag
func TagGetFrameTrackNumber(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Track number/Position in set"))) > 0 {
		for _, frameText := range tag.GetFrames(tag.CommonID("Track number/Position in set")) {
			text, ok := frameText.(id3v2.TextFrame)
			if ok {
				return text.Text
			}
		}
	}
	return ""
}

// TagGetFrameTrackTotals : get total tracks number frame from input Tag
func TagGetFrameTrackTotals(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "trackTotals" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagGetFrameArtwork : get artwork frame from input Tag
func TagGetFrameArtwork(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Attached picture"))) > 0 {
		for _, framePicture := range tag.GetFrames(tag.CommonID("Attached picture")) {
			picture, ok := framePicture.(id3v2.PictureFrame)
			if ok {
				return string(picture.Picture)
			}
		}
	}
	return ""
}

// TagGetFrameArtworkURL : get artwork URL frame from input Tag
func TagGetFrameArtworkURL(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "artwork" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagGetFrameLyrics : get lyrics frame from input Tag
func TagGetFrameLyrics(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))) > 0 {
		for _, frameLyrics := range tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription")) {
			lyrics, ok := frameLyrics.(id3v2.UnsynchronisedLyricsFrame)
			if ok {
				return lyrics.Lyrics
			}
		}
	}
	return ""
}

// TagGetFrameOrigin : get origin frame from input Tag
func TagGetFrameOrigin(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "origin" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagGetFrameDuration : get duration frame from input Tag
func TagGetFrameDuration(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "duration" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagGetFrameSpotifyID : get Spotify ID frame from input Tag
func TagGetFrameSpotifyID(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "spotifyid" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagHasFrame : return True if open input Tag has valued input frame
func TagHasFrame(tag *id3v2.Tag, frame int) bool {
	return TagGetFrame(tag, frame) != ""
}

// GetID3Frame : get Track ID3 input frame string value
func (track Track) GetID3Frame(frame int) string {
	tag, err := id3v2.Open(track.Filename(), id3v2.Options{Parse: true})
	if tag == nil || err != nil {
		return ""
	}
	defer tag.Close()
	return TagGetFrame(tag, frame)
}

// HasID3Frame : return True if Track has input ID3 frame
func (track *Track) HasID3Frame(frame int) bool {
	return track.GetID3Frame(frame) != ""
}
