package processor

import (
	"errors"
	"fmt"

	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys/cmd"
)

type normalizer struct{}

func (normalizer) Applies(object interface{}) bool {
	_, ok := object.(*entity.Track)
	return ok
}

// ErrLoudnessSkipped signals that loudness normalization was skipped for a
// track: the rest of the processing chain still applies and the sync goes on.
var ErrLoudnessSkipped = errors.New("loudness normalization skipped")

func (normalizer) Do(object interface{}) error {
	track, ok := object.(*entity.Track)
	if !ok {
		return errors.New("processor does not support such object")
	}

	loudness, err := cmd.FFmpeg().LoudnessDetect(track.Path().Download())
	if err != nil {
		// loudness is a nice-to-have: a failure must not
		// discard a track with audio and artwork ready
		return fmt.Errorf("%w: %v", ErrLoudnessSkipped, err)
	}
	return cmd.FFmpeg().LoudnessNormalize(track.Path().Download(), loudness)
}
