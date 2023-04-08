package processor

import (
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util/cmd"
)

type normalizer struct {
	Processor
}

func (normalizer) Do(track *entity.Track) error {
	volumeDelta, err := cmd.FFmpeg().VolumeDetect(track.Path().Download())
	if err != nil {
		return err
	}

	return cmd.FFmpeg().VolumeAdd(track.Path().Download(), volumeDelta)
}
