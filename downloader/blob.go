package downloader

import (
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/streambinder/spotitube/processor"
	"github.com/streambinder/spotitube/sys"
)

type blob struct {
	Downloader
}

func init() {
	downloaders = append(downloaders, blob{})
}

func (blob) supports(url string) bool {
	response, err := http.Head(url) // nolint
	if err != nil {
		return false
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return false
	}

	switch response.Header.Get("Content-Type") {
	case "image/jpeg":
		return true
	default:
		return false
	}
}

func (blob) download(url, path string, processor processor.Processor, channels ...chan []byte) error {
	response, err := http.Get(url) // nolint
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.New("cannot get blob: " + response.Status)
	}

	output, err := os.Create(path)
	if err != nil {
		return err
	}
	defer output.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if processor != nil && processor.Applies(body) {
		if err := processor.Do(body); err != nil {
			return err
		}
	}

	for _, ch := range channels {
		ch <- body
	}

	return sys.ErrOnly(output.Write(body))
}
