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
	"github.com/streambinder/spotitube/util"
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

	response, err := http.DefaultClient.Do(request)
	if err != nil && errors.Is(err, context.Canceled) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == 404 {
		return nil, nil
	} else if response.StatusCode == 429 {
		util.SleepUntilRetry(response.Header)
		return composer.get(url, ctx)
	} else if response.StatusCode != 200 {
		return nil, errors.New("cannot fetch results on lyrics.ovh: " + response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	entry := new(ovhResponse)
	if err := json.Unmarshal(body, entry); err != nil {
		return nil, err
	}

	return []byte(entry.Lyrics), nil
}
