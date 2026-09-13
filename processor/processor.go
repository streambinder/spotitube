package processor

type Processor interface {
	Do(interface{}) error
	Applies(interface{}) bool
}

func Do(object interface{}) error {
	for _, processor := range []Processor{
		Artwork{},
		normalizer{},
		encoder{},
	} {
		if supported := processor.Applies(object); supported {
			if err := processor.Do(object); err != nil {
				return err
			}
		}
	}
	return nil
}
