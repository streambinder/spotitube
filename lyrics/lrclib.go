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

type lrclib struct {
	Composer
}

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

func (composer lrclib) get(url string, ctxs ...context.Context) ([]byte, error) {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil && errors.Is(err, context.Canceled) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	switch {
	case response.StatusCode == 404:
		return nil, nil
	case response.StatusCode == 429:
		sys.SleepUntilRetry(response.Header)
		return composer.get(url, ctx)
	case response.StatusCode != 200:
		return nil, errors.New("cannot fetch results on lrclib: " + response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	entry := new(lrclibResponse)
	if err := json.Unmarshal(body, entry); err != nil {
		return nil, err
	}

	lyrics := entry.PlainLyrics
	if len(entry.SyncedLyrics) > 0 {
		lyrics = entry.SyncedLyrics
	}
	return []byte(regexp.MustCompile(`\[(\d{2}:\d{2}\.\d{2})\]\s+`).ReplaceAllString(lyrics, `[$1]`)), nil
}
