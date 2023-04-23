package processor

import (
	"errors"

	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util/cmd"
)

type normalizer struct {
	Processor
}

func (normalizer) Applies(object interface{}) bool {
	_, ok := object.(*entity.Track)
	return ok
}

func (normalizer) Do(object interface{}) error {
	track, ok := object.(*entity.Track)
	if !ok {
		return errors.New("processor does not support such object")
	}

	volumeDelta, err := cmd.FFmpeg().VolumeDetect(track.Path().Download())
	if err != nil {
		return err
	}

	return cmd.FFmpeg().VolumeAdd(track.Path().Download(), volumeDelta)
}
