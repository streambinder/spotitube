package processor

import "errors"

type Processor interface {
	Do(interface{}) error
	Applies(interface{}) bool
}

func Do(object interface{}) error {
	var skipped error
	for _, processor := range []Processor{
		Artwork{},
		normalizer{},
		encoder{},
	} {
		if supported := processor.Applies(object); supported {
			if err := processor.Do(object); err != nil {
				// a skipped normalization applies to this processor
				// only: the rest of the chain still runs
				if errors.Is(err, ErrLoudnessSkipped) {
					skipped = err
					continue
				}
				return err
			}
		}
	}
	return skipped
}
