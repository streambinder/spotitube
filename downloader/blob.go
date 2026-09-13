package downloader

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/streambinder/spotitube/processor"
	"github.com/streambinder/spotitube/sys"
)

type blob struct {
	Downloader
}

const mimeJPEG = "image/jpeg"

// a slow host must not hang the download forever on the default
// client: cap every blob request at 30 seconds
var httpClient = &http.Client{Timeout: 30 * time.Second}

func init() {
	downloaders = append(downloaders, blob{})
}

func (blob) supports(ctx context.Context, url string) bool {
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return false
	}

	switch response.Header.Get("Content-Type") {
	case mimeJPEG, "audio/mpeg":
		return true
	default:
		return false
	}
}

func (blob) download(ctx context.Context, url, path string, processor processor.Processor, channels ...chan []byte) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	response, err := httpClient.Do(request)
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

	if processor != nil && processor.Applies(&body) {
		if err := processor.Do(&body); err != nil {
			return err
		}
	}

	for _, ch := range channels {
		ch <- body
	}

	return sys.ErrOnly(output.Write(body))
}
