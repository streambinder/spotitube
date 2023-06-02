package lyrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agnivade/levenshtein"
	jsoniter "github.com/json-iterator/go"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
)

type genius struct {
	Composer
}

type geniusSearch struct {
	Response struct {
		Hits []struct {
			Result struct {
				URL    string
				Title  string
				Artist struct {
					Name string
				} `json:"primary_artist"`
			}
		}
	}
}

func init() {
	composers = append(composers, &genius{})
}

func (composer genius) search(track *entity.Track, ctxs ...context.Context) ([]byte, error) {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}

	query := track.Song()
	for _, artist := range track.Artists {
		query = fmt.Sprintf("%s %s", query, artist)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.genius.com/search?q=%s", url.QueryEscape(query)), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("GENIUS_TOKEN")))

	response, err := http.DefaultClient.Do(request)
	if err != nil && errors.Is(err, context.Canceled) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == 429 {
		util.SleepUntilRetry(response.Header)
		return composer.search(track, ctx)
	} else if response.StatusCode != 200 {
		return nil, errors.New("cannot search lyrics on genius: " + response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var data geniusSearch
	if err := jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var (
		geniusURL   string
		minDistance = 50
	)
	for _, hit := range data.Response.Hits {
		var (
			urlCompliant    = strings.HasPrefix(hit.Result.URL, "https://genius.com/")
			titleCompliant  = strings.Contains(util.Flatten(hit.Result.Title), util.Flatten(track.Song()))
			artistCompliant = strings.Contains(util.Flatten(hit.Result.Artist.Name), util.Flatten(track.Artists[0]))
			distance        = levenshtein.ComputeDistance(
				util.UniqueFields(query),
				util.UniqueFields(fmt.Sprintf("%s %s", hit.Result.Title, hit.Result.Artist.Name)),
			)
		)
		if urlCompliant && titleCompliant && artistCompliant && distance < minDistance {
			geniusURL = hit.Result.URL
			minDistance = distance
		}
	}

	if geniusURL == "" {
		return nil, nil
	}

	return composer.fromGeniusURL(geniusURL, ctx)
}

func (composer genius) fromGeniusURL(url string, ctx context.Context) ([]byte, error) {
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

	if response.StatusCode == 429 {
		util.SleepUntilRetry(response.Header)
		return composer.fromGeniusURL(url, ctx)
	} else if response.StatusCode != 200 {
		return nil, errors.New("cannot fetch lyrics on genius: " + response.Status)
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, err
	}

	var data []byte
	document.Find("div[data-lyrics-container='true']").Contents().
		Each(documentParser(&data))

	return data, nil
}

func documentParser(data *[]byte) func(i int, s *goquery.Selection) {
	return func(i int, s *goquery.Selection) {
		switch goquery.NodeName(s) {
		case "br", "div":
			*data = append(*data, 10)
		case "#text":
			*data = append(*data, []byte(s.Text())...)
		default:
			s.Contents().Each(documentParser(data))
		}
	}
}
