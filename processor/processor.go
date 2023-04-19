package processor

type Processor interface {
	do(interface{}) error
	applies(interface{}) bool
}

func Do(object interface{}) error {
	for _, processor := range []Processor{
		artwork{},
		normalizer{},
		encoder{},
	} {
		if supported, err := processor.applies(object), processor.do(object); supported && err != nil {
			return err
		}
	}
	return nil
}
