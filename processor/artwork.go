package processor

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"

	"github.com/nfnt/resize"
)

type Artwork struct {
	Processor
}

func (Artwork) Applies(object interface{}) bool {
	_, ok := object.(*[]byte)
	return ok
}

func (Artwork) Do(object interface{}) error {
	data, ok := object.(*[]byte)
	if !ok {
		return errors.New("processor does not support such object")
	}

	img, _, err := image.Decode(bytes.NewReader(*data))
	if err != nil {
		return err
	}

	var resized bytes.Buffer
	if err := jpeg.Encode(&resized, resize.Resize(300, 0, img, resize.Lanczos3), nil); err != nil {
		return err
	}

	*data = resized.Bytes()
	return nil
}
