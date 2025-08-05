package id3

import (
	"fmt"
	"strings"

	"github.com/streambinder/id3v2-sylt"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/sys"
)

const (
	frameAttachedPicture      = "Attached picture"
	frameTrackNumber          = "Track number/Position in set"
	frameUnsynchronizedLyrics = "Unsynchronised lyrics/text transcription"
	frameSynchronizedLyrics   = "Synchronised lyrics/text"
	frameSpotifyID            = "Spotify ID"
	frameArtworkURL           = "Artwork URL"
	frameDuration             = "Duration"
	frameUpstreamURL          = "Upstream URL"
)

type Tag struct {
	id3v2.Tag
	Cache map[string]string
}

func Open(path string, options id3v2.Options) (*Tag, error) {
	tag, err := id3v2.Open(path, options)
	if err != nil {
		return nil, err
	}
	return &Tag{*tag, make(map[string]string)}, err
}

func (tag *Tag) SetTrackNumber(number string) {
	tag.AddFrame(
		tag.CommonID(frameTrackNumber),
		id3v2.TextFrame{
			Encoding: tag.DefaultEncoding(),
			Text:     number,
		},
	)
}

func (tag *Tag) TrackNumber() string {
	return tag.GetTextFrame(tag.CommonID(frameTrackNumber)).Text
}

func (tag *Tag) setUserDefinedText(key, value string) {
	tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
		Encoding:    tag.DefaultEncoding(),
		Description: key,
		Value:       value,
	})
}

func (tag *Tag) userDefinedText(key string) string {
	if value, ok := tag.Cache[key]; ok {
		return value
	}

	for _, frame := range tag.GetFrames(tag.CommonID("User defined text information frame")) {
		frame, ok := frame.(id3v2.UserDefinedTextFrame)
		if ok {
			tag.Cache[frame.UniqueIdentifier()] = frame.Value
		}

		if strings.EqualFold(frame.UniqueIdentifier(), key) {
			return frame.Value
		}
	}

	return ""
}

func (tag *Tag) SetSpotifyID(id string) {
	tag.setUserDefinedText(frameSpotifyID, id)
}

func (tag *Tag) SpotifyID() string {
	return tag.userDefinedText(frameSpotifyID)
}

func (tag *Tag) SetArtworkURL(url string) {
	tag.setUserDefinedText(frameArtworkURL, url)
}

func (tag *Tag) ArtworkURL() string {
	return tag.userDefinedText(frameArtworkURL)
}

func (tag *Tag) SetDuration(duration string) {
	tag.setUserDefinedText(frameDuration, duration)
}

func (tag *Tag) Duration() string {
	return tag.userDefinedText(frameDuration)
}

func (tag *Tag) SetUpstreamURL(url string) {
	tag.setUserDefinedText(frameUpstreamURL, url)
}

func (tag *Tag) UpstreamURL() string {
	return tag.userDefinedText(frameUpstreamURL)
}

func (tag *Tag) SetAttachedPicture(picture []byte) {
	tag.AddAttachedPicture(id3v2.PictureFrame{
		Encoding:    tag.DefaultEncoding(),
		MimeType:    "image/jpeg",
		PictureType: id3v2.PTFrontCover,
		Description: "Front cover",
		Picture:     picture,
	})
}

func (tag *Tag) AttachedPicture() (string, []byte) {
	frame, ok := tag.GetLastFrame(tag.CommonID(frameAttachedPicture)).(id3v2.PictureFrame)
	if ok {
		return frame.MimeType, frame.Picture
	}
	return "", []byte{}
}

func (tag *Tag) SetLyrics(title, lyrics string) {
	tag.setSynchronizedLyrics(title, lyrics)
	tag.setUnsynchronizedLyrics(title, lyrics)
}

func (tag *Tag) setSynchronizedLyrics(title, data string) {
	if !lyrics.IsSynced(data) {
		return
	}

	var syncedText []id3v2.SyncedText
	for _, line := range lyrics.GetSync(data) {
		syncedText = append(syncedText, id3v2.SyncedText{
			Timestamp: line.Time,
			Text:      line.Text,
		})
	}

	tag.AddSynchronisedLyricsFrame(id3v2.SynchronisedLyricsFrame{
		Encoding:          tag.DefaultEncoding(),
		Language:          "eng",
		TimestampFormat:   id3v2.SYLTAbsoluteMillisecondsTimestampFormat,
		ContentType:       id3v2.SYLTLyricsContentType,
		ContentDescriptor: title,
		SynchronizedTexts: syncedText,
	})
}

func (tag *Tag) SynchronizedLyrics() string {
	frame, ok := tag.GetLastFrame(tag.CommonID(frameSynchronizedLyrics)).(id3v2.SynchronisedLyricsFrame)
	if ok {
		var lyrics []string
		for _, line := range frame.SynchronizedTexts {
			lyrics = append(lyrics, fmt.Sprintf("[%s]%s", sys.MillisToColonMinutes(line.Timestamp), line.Text))
		}
		return strings.Join(lyrics, "\n")
	}
	return ""
}

func (tag *Tag) setUnsynchronizedLyrics(title, data string) {
	tag.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
		Encoding:          tag.DefaultEncoding(),
		Language:          "eng",
		ContentDescriptor: title,
		Lyrics:            lyrics.GetPlain(data),
	})
}

func (tag *Tag) UnsynchronizedLyrics() string {
	frame, ok := tag.GetLastFrame(tag.CommonID(frameUnsynchronizedLyrics)).(id3v2.UnsynchronisedLyricsFrame)
	if ok {
		return frame.Lyrics
	}
	return ""
}

func (tag *Tag) Close() error {
	if err := tag.Tag.Close(); err != id3v2.ErrNoFile && err != nil {
		return err
	}
	return nil
}
