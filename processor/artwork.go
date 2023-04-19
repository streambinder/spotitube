package processor

import (
	"bufio"
	"bytes"
	"errors"
	"image"
	"image/jpeg"

	"github.com/nfnt/resize"
)

type artwork struct {
	Processor
}

func (artwork) applies(object interface{}) bool {
	_, ok := object.(*[]byte)
	return ok
}

func (artwork) do(object interface{}) error {
	data, ok := object.(*[]byte)
	if !ok {
		return errors.New("processor does not support such object")
	}

	image, _, err := image.Decode(bytes.NewReader(*data))
	if err != nil {
		return err
	}

	var resized bytes.Buffer
	if err := jpeg.Encode(bufio.NewWriter(&resized), resize.Resize(300, 0, image, resize.Lanczos3), nil); err != nil {
		return err
	}

	*data = resized.Bytes()
	return nil
}
