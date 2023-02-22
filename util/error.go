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
