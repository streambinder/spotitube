package processor

import (
	"github.com/streambinder/spotitube/entity"
)

type Processor interface {
	Do(*entity.Track) error
}

func Do(track *entity.Track) error {
	for _, processor := range []Processor{
		normalizer{},
		encoder{},
	} {
		if err := processor.Do(track); err != nil {
			return err
		}
	}
	return nil
}
