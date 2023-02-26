package util

func ErrWrap[T any](def T) func(T, error) T {
	return func(value T, err error) T {
		if err != nil {
			value = def
		}
		return value
	}
}

func ErrOnly(parameters ...any) error {
	if len(parameters) == 0 {
		return nil
	}

	err := parameters[len(parameters)-1]
	if err == nil {
		return nil
	}

	return err.(error)
}
