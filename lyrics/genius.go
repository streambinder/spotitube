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
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
	"github.com/tidwall/gjson"
)

type genius struct {
	Composer
}

func init() {
	composers = append(composers, &genius{})
}

func (composer genius) search(track *entity.Track, ctxs ...context.Context) ([]byte, error) {
	ctx := context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}

	query := track.Title
	for _, artist := range track.Artists {
		query = fmt.Sprintf("%s %s", query, artist)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.genius.com/search?q=%s", url.QueryEscape(query)), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("GENIUS_TOKEN")))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
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

	var (
		geniusURL   string
		minDistance = 50
	)
	gjson.Get(string(body), "response.hits").ForEach(func(key, value gjson.Result) bool {
		var (
			url             = gjson.Get(value.String(), "result.url").String()
			urlCompliant    = strings.HasPrefix(url, "https://genius.com/")
			title           = gjson.Get(value.String(), "result.title").String()
			titleCompliant  = strings.Contains(util.Flatten(title), util.Flatten(track.Title))
			artist          = gjson.Get(value.String(), "result.primary_artist.name").String()
			artistCompliant = strings.Contains(util.Flatten(artist), util.Flatten(track.Artists[0]))
			distance        = levenshtein.ComputeDistance(
				util.UniqueFields(query),
				util.UniqueFields(fmt.Sprintf("%s %s", title, artist)),
			)
		)
		if urlCompliant && titleCompliant && artistCompliant && distance < minDistance {
			geniusURL = url
			minDistance = distance
		}

		return true
	})

	if geniusURL != "" {
		return composer.fromGeniusURL(geniusURL, ctx)
	}

	return nil, nil
}

func (composer genius) fromGeniusURL(url string, ctx context.Context) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
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
