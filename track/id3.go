package track

import (
	"strconv"
	"strings"

	"github.com/bogem/id3v2"
)

const (
	// ID3FrameTitle is the ID3 title frame tag identifier
	ID3FrameTitle = iota
	// ID3FrameSong is the ID3 song frame tag identifier
	ID3FrameSong
	// ID3FrameArtist is the ID3 artist frame tag identifier
	ID3FrameArtist
	// ID3FrameAlbum is the ID3 album frame tag identifier
	ID3FrameAlbum
	// ID3FrameGenre is the ID3 genre frame tag identifier
	ID3FrameGenre
	// ID3FrameYear is the ID3 year frame tag identifier
	ID3FrameYear
	// ID3FrameFeaturings is the ID3 featurings frame tag identifier
	ID3FrameFeaturings
	// ID3FrameTrackNumber is the ID3 track number frame tag identifier
	ID3FrameTrackNumber
	// ID3FrameTrackTotals is the ID3 total tracks number frame tag identifier
	ID3FrameTrackTotals
	// ID3FrameArtwork is the ID3 artwork frame tag identifier
	ID3FrameArtwork
	// ID3FrameArtworkURL is the ID3 artwork URL frame tag identifier
	ID3FrameArtworkURL
	// ID3FrameLyrics is the ID3 lyrics frame tag identifier
	ID3FrameLyrics
	// ID3FrameOrigin is the ID3 origin frame tag identifier
	ID3FrameOrigin
	// ID3FrameDuration is the ID3 duration frame tag identifier
	ID3FrameDuration
	// ID3FrameSpotifyID is the ID3 Spotify ID frame tag identifier
	ID3FrameSpotifyID
)

// Flush persists tracks frames into given open Tag
func (track Track) Flush() error {
	tag, err := id3v2.Open(track.FilenameTemporary(), id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer tag.Close()
	defer tag.Save()

	// official metadata fields
	tag.SetTitle(track.Title)
	tag.SetArtist(track.Artist)
	tag.SetAlbum(track.Album)
	tag.SetGenre(track.Genre)
	tag.SetYear(track.Year)
	tag.AddFrame(
		tag.CommonID("Track number/Position in set"),
		id3v2.TextFrame{
			Encoding: id3v2.EncodingUTF8,
			Text:     strconv.Itoa(track.TrackNumber),
		},
	)
	tag.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
		Encoding:          id3v2.EncodingUTF8,
		Language:          "eng",
		ContentDescriptor: track.Title,
		Lyrics:            track.Lyrics,
	})
	tag.AddAttachedPicture(id3v2.PictureFrame{
		Encoding:    id3v2.EncodingUTF8,
		MimeType:    "image/jpeg",
		PictureType: id3v2.PTFrontCover,
		Description: "Front cover",
		Picture:     *track.Artwork,
	})

	// unofficial metadata fields
	for key, value := range map[string]string{
		"song":        track.Song,
		"featurings":  strings.Join(track.Featurings, "|"),
		"trackTotals": strconv.Itoa(track.TrackTotals),
		"artwork":     track.ArtworkURL,
		"origin":      track.URL,
		"duration":    strconv.Itoa(track.Duration),
		"spotifyid":   track.SpotifyID,
	} {
		tag.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: key,
			Text:        value,
		})
	}

	return nil
}

// GetTag opens, parses and returns given path's given frame tag
func GetTag(path string, frame int) string {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return ""
	}
	defer tag.Close()

	return TagGetFrame(tag, frame)
}

// TagGetFrame gets given frame from given open Tag
func TagGetFrame(tag *id3v2.Tag, frame int) string {
	switch frame {
	case ID3FrameTitle:
		return tag.Title()
	case ID3FrameSong:
		return tagGetFrameSong(tag)
	case ID3FrameArtist:
		return tag.Artist()
	case ID3FrameAlbum:
		return tag.Album()
	case ID3FrameGenre:
		return tag.Genre()
	case ID3FrameYear:
		return tag.Year()
	case ID3FrameFeaturings:
		return tagGetFrameFeaturings(tag)
	case ID3FrameTrackNumber:
		return tagGetFrameTrackNumber(tag)
	case ID3FrameTrackTotals:
		return tagGetFrameTrackTotals(tag)
	case ID3FrameArtwork:
		return tagGetFrameArtwork(tag)
	case ID3FrameArtworkURL:
		return tagGetFrameArtworkURL(tag)
	case ID3FrameLyrics:
		return tagGetFrameLyrics(tag)
	case ID3FrameOrigin:
		return tagGetFrameOrigin(tag)
	case ID3FrameDuration:
		return tagGetFrameDuration(tag)
	case ID3FrameSpotifyID:
		return tagGetFrameSpotifyID(tag)
	}
	return ""
}

func tagGetFrameSong(tag *id3v2.Tag) string {
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

func tagGetFrameFeaturings(tag *id3v2.Tag) string {
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

func tagGetFrameTrackNumber(tag *id3v2.Tag) string {
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

func tagGetFrameTrackTotals(tag *id3v2.Tag) string {
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

func tagGetFrameArtwork(tag *id3v2.Tag) string {
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

func tagGetFrameArtworkURL(tag *id3v2.Tag) string {
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

func tagGetFrameLyrics(tag *id3v2.Tag) string {
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

func tagGetFrameOrigin(tag *id3v2.Tag) string {
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

func tagGetFrameDuration(tag *id3v2.Tag) string {
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

func tagGetFrameSpotifyID(tag *id3v2.Tag) string {
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

func (track Track) getID3Frame(frame int) string {
	tag, err := id3v2.Open(track.Filename(), id3v2.Options{Parse: true})
	if tag == nil || err != nil {
		return ""
	}
	defer tag.Close()
	return TagGetFrame(tag, frame)
}
