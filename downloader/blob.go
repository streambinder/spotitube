package downloader

import (
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/streambinder/spotitube/util"
)

type blob struct {
	Downloader
}

func init() {
	downloaders = append(downloaders, blob{})
}

func (blob) Supports(url string) bool {
	response, err := http.Head(url)
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

func (blob) Download(url, path string, channels ...chan []byte) error {
	response, err := http.Get(url)
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

	for _, ch := range channels {
		ch <- body
	}

	return util.ErrOnly(output.Write(body))
}
