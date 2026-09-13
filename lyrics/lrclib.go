package lyrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys"
)

var reLrclibWhitespace = regexp.MustCompile(`\[(\d{2}:\d{2}\.\d{2})\]\s+`)

type lrclib struct{}

type lrclibResponse struct {
	SyncedLyrics string `json:"syncedLyrics"`
	PlainLyrics  string `json:"plainLyrics"`
}

func init() {
	composers = append(composers, &lrclib{})
}

func (composer lrclib) search(track *entity.Track, ctxs ...context.Context) ([]byte, error) {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}

	return composer.get(fmt.Sprintf("https://lrclib.net/api/get?artist_name=%s&track_name=%s",
		url.QueryEscape(track.Artists[0]),
		url.QueryEscape(track.Title)), ctx)
}

type lrclibStatusAction int

const (
	lrclibStatusProceed lrclibStatusAction = iota
	lrclibStatusNotFound
	lrclibStatusRetry
)

// classifyStatus maps a response status code to the action to take.
// Server errors are retried only once, unlike rate limiting.
func classifyStatus(response *http.Response, serverErrs *int) (lrclibStatusAction, error) {
	switch {
	case response.StatusCode == 404:
		return lrclibStatusNotFound, nil
	case response.StatusCode == 429:
		sys.SleepUntilRetry(response.Header)
		return lrclibStatusRetry, nil
	case response.StatusCode >= 500 && response.StatusCode <= 599:
		if *serverErrs > 0 {
			return lrclibStatusProceed, errors.New("cannot fetch results on lrclib: " + response.Status)
		}
		*serverErrs++
		sys.SleepUntilRetry(response.Header)
		return lrclibStatusRetry, nil
	default:
		if response.StatusCode != 200 {
			return lrclibStatusProceed, errors.New("cannot fetch results on lrclib: " + response.Status)
		}
		return lrclibStatusProceed, nil
	}
}

func (composer lrclib) get(url string, ctxs ...context.Context) ([]byte, error) {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	serverErrs := 0
	for attempt := 0; attempt < sys.MaxRetries; attempt++ {
		result, retry, getErr := func() ([]byte, bool, error) {
			response, err := http.DefaultClient.Do(request)
			if err != nil && errors.Is(err, context.Canceled) {
				return nil, false, nil
			} else if err != nil {
				return nil, false, err
			}
			defer response.Body.Close()

			action, actionErr := classifyStatus(response, &serverErrs)
			if actionErr != nil {
				return nil, false, actionErr
			}
			if action == lrclibStatusNotFound {
				return nil, false, nil
			}
			if action == lrclibStatusRetry {
				return nil, true, nil
			}

			body, readErr := io.ReadAll(response.Body)
			if readErr != nil {
				return nil, false, readErr
			}

			entry := new(lrclibResponse)
			if unmarshalErr := json.Unmarshal(body, entry); unmarshalErr != nil {
				return nil, false, unmarshalErr
			}

			lyrics := entry.PlainLyrics
			if len(entry.SyncedLyrics) > 0 {
				lyrics = entry.SyncedLyrics
			}
			return []byte(reLrclibWhitespace.ReplaceAllString(lyrics, `[$1]`)), false, nil
		}()
		if retry {
			// rebuild request since body was consumed
			var reqErr error
			request, reqErr = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if reqErr != nil {
				return nil, reqErr
			}
			continue
		}
		return result, getErr
	}
	return nil, errors.New("lrclib: max retries exceeded")
}
