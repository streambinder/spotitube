package processor

import (
	"errors"
	"strconv"

	"github.com/streambinder/id3v2-sylt"
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

	tag, err := id3.Open(track.Path().Download(), id3v2.Options{Parse: false})
	if err != nil {
		return err
	}
	defer tag.Close()

	tag.SetSpotifyID(track.ID)
	tag.SetTitle(track.Title)
	tag.SetArtist(track.Artists[0])
	tag.SetAlbum(track.Album)
	tag.SetArtworkURL(track.Artwork.URL)
	tag.SetAttachedPicture(track.Artwork.Data)
	tag.SetDuration(strconv.Itoa(track.Duration))
	tag.SetLyrics(track.Title, track.Lyrics)
	tag.SetTrackNumber(strconv.Itoa(track.Number))
	tag.SetYear(strconv.Itoa(track.Year))
	tag.SetUpstreamURL(track.UpstreamURL)
	return tag.Save()
}
