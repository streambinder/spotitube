package util

func ErrWrap[T any](def T) func(T, error) T {
	return func(value T, err error) T {
		if err != nil {
			value = def
		}
		return value
	}
}

func ErrOnly[T any](data T, err error) error {
	return err
}

func ErrSuppress(err error, types ...error) error {
	for _, errType := range types {
		if err == errType {
			return nil
		}
	}

	if len(types) > 0 {
		return err
	}

	return nil
}
