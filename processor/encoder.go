package processor

import (
	"errors"
	"strconv"

	"github.com/bogem/id3v2/v2"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/entity/id3"
)

type encoder struct {
	Processor
}

func (encoder) Applies(object interface{}) bool {
	_, ok := object.(*entity.Track)
	return ok
}

func (encoder) Do(object interface{}) error {
	track, ok := object.(*entity.Track)
	if !ok {
		return errors.New("processor does not support such object")
	}

	tag, err := id3v2.Open(
		track.Path().Download(),
		id3v2.Options{Parse: false})
	if err != nil {
		return err
	}
	defer tag.Close()

	tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
		Encoding:    tag.DefaultEncoding(),
		Description: id3.FrameSpotifyID,
		Value:       track.ID,
	})
	tag.SetTitle(track.Title)
	tag.SetArtist(track.Artists[0])
	tag.SetAlbum(track.Album)
	tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
		Encoding:    tag.DefaultEncoding(),
		Description: id3.FrameArtworkURL,
		Value:       track.Artwork.URL,
	})
	tag.AddAttachedPicture(id3v2.PictureFrame{
		Encoding:    tag.DefaultEncoding(),
		MimeType:    "image/jpeg",
		PictureType: id3v2.PTFrontCover,
		Description: "Front cover",
		Picture:     track.Artwork.Data,
	})
	tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
		Encoding:    tag.DefaultEncoding(),
		Description: id3.FrameDuration,
		Value:       strconv.Itoa(track.Duration),
	})
	tag.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
		Encoding:          tag.DefaultEncoding(),
		Language:          "eng",
		ContentDescriptor: track.Title,
		Lyrics:            track.Lyrics,
	})
	tag.AddFrame(
		tag.CommonID("Track number/Position in set"),
		id3v2.TextFrame{
			Encoding: tag.DefaultEncoding(),
			Text:     strconv.Itoa(track.Number),
		},
	)
	tag.SetYear(strconv.Itoa(track.Year))
	tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
		Encoding:    tag.DefaultEncoding(),
		Description: id3.FrameUpstreamURL,
		Value:       track.UpstreamURL,
	})

	return tag.Save()
}
