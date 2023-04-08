package processor

import (
	"strconv"

	"github.com/bogem/id3v2/v2"
	"github.com/streambinder/spotitube/entity"
)

type encoder struct {
	Processor
}

func (encoder) Do(track *entity.Track) error {
	tag, err := id3v2.Open(
		track.Path().Download(),
		id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer tag.Close()

	tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
		Encoding:    tag.DefaultEncoding(),
		Description: "Spotify ID",
		Value:       track.ID,
	})
	tag.SetTitle(track.Title)
	tag.SetArtist(track.Artists[0])
	tag.SetAlbum(track.Album)
	tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
		Encoding:    tag.DefaultEncoding(),
		Description: "Artwork URL",
		Value:       track.ArtworkURL,
	})
	tag.AddAttachedPicture(id3v2.PictureFrame{
		Encoding:    tag.DefaultEncoding(),
		MimeType:    "image/jpeg",
		PictureType: id3v2.PTFrontCover,
		Description: "Front cover",
		Picture:     track.Artwork,
	})
	tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
		Encoding:    tag.DefaultEncoding(),
		Description: "Duration",
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
	tag.SetYear(track.Year)
	tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
		Encoding:    tag.DefaultEncoding(),
		Description: "Upstream URL",
		Value:       track.UpstreamURL,
	})

	return tag.Save()
}
