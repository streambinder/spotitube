package lyrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys"
)

type lyricsOvh struct {
	Composer
}

type ovhResponse struct {
	Lyrics string `json:"lyrics"`
}

func init() {
	// composers = append(composers, &lyricsOvh{})
}

func (composer lyricsOvh) search(track *entity.Track, ctxs ...context.Context) ([]byte, error) {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}

	return composer.get(fmt.Sprintf("https://api.lyrics.ovh/v1/%s/%s",
		url.QueryEscape(track.Artists[0]),
		url.QueryEscape(track.Title)), ctx)
}

func (composer lyricsOvh) get(url string, ctxs ...context.Context) ([]byte, error) {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < sys.MaxRetries; attempt++ {
		result, retry, getErr := func() ([]byte, bool, error) {
			response, err := http.DefaultClient.Do(request)
			if err != nil && errors.Is(err, context.Canceled) {
				return nil, false, nil
			} else if err != nil {
				return nil, false, err
			}
			defer response.Body.Close()

			switch {
			case response.StatusCode == 404:
				return nil, false, nil
			case response.StatusCode == 429:
				sys.SleepUntilRetry(response.Header)
				return nil, true, nil
			case response.StatusCode != 200:
				return nil, false, errors.New("cannot fetch results on lyrics.ovh: " + response.Status)
			}

			body, readErr := io.ReadAll(response.Body)
			if readErr != nil {
				return nil, false, readErr
			}

			entry := new(ovhResponse)
			if unmarshalErr := json.Unmarshal(body, entry); unmarshalErr != nil {
				return nil, false, unmarshalErr
			}
			return []byte(entry.Lyrics), false, nil
		}()
		if retry {
			// rebuild request since body was consumed
			request, _ = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			continue
		}
		return result, getErr
	}
	return nil, errors.New("lyrics.ovh: max retries exceeded")
}
