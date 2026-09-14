package processor

import (
	"errors"

	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys/cmd"
)

type normalizer struct{}

func (normalizer) Applies(object interface{}) bool {
	_, ok := object.(*entity.Track)
	return ok
}

func (normalizer) Do(object interface{}) error {
	track, ok := object.(*entity.Track)
	if !ok {
		return errors.New("processor does not support such object")
	}

	loudness, err := cmd.FFmpeg().LoudnessDetect(track.Path().Download())
	if err != nil {
		return err
	}
	return cmd.FFmpeg().LoudnessNormalize(track.Path().Download(), loudness)
}
