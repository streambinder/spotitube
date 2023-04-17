package processor

import (
	"github.com/streambinder/spotitube/entity"
)

type Processor interface {
	do(*entity.Track) error
}

func Do(track *entity.Track) error {
	for _, processor := range []Processor{
		normalizer{},
		encoder{},
	} {
		if err := processor.do(track); err != nil {
			return err
		}
	}
	return nil
}
