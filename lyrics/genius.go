package lyrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
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

const contextValueLabelMainArtist = "mainArtistOnly"

var fallbackGeniusToken = ""

type contextValueLabel string

type genius struct {
	Composer
}

type geniusSearch struct {
	Response struct {
		Hits []struct {
			Result geniusResult
		}
	}
}

type geniusResult struct {
	track  *entity.Track
	query  string // not part of APIs
	URL    string
	Title  string
	Artist struct {
		Name string
	} `json:"primary_artist"`
}

func init() {
	composers = append(composers, &genius{})
}

func (composer genius) search(track *entity.Track, ctxs ...context.Context) ([]byte, error) {
	var (
		ctx            = context.Background()
		mainArtistOnly = false
	)
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}
	if ctxMainArtistOnly, ok := ctx.Value(contextValueLabel(contextValueLabelMainArtist)).(bool); ok {
		mainArtistOnly = ctxMainArtistOnly
	}

	query := track.Song()
	for _, artist := range track.Artists {
		query = fmt.Sprintf("%s %s", query, artist)
		if mainArtistOnly {
			break
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.genius.com/search?q=%s", url.QueryEscape(query)), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", util.Fallback(os.Getenv("GENIUS_TOKEN"), fallbackGeniusToken)))

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
		score int
		url   string
	)
	for _, hit := range data.Response.Hits {
		hit.Result.track = track
		hit.Result.query = query
		if hitScore := hit.Result.score(); hit.Result.compliant() && hitScore > score {
			url = hit.Result.URL
			score = hitScore
		}
	}

	if url == "" && mainArtistOnly {
		return nil, nil
	} else if url == "" {
		return composer.search(
			track, context.WithValue(ctx, contextValueLabel(contextValueLabelMainArtist), true))
	}

	return composer.fromGeniusURL(url, ctx)
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

// compliance check works as a barrier before checking on the result score
// so to ensure that only the results that pass certain pre-checks get returned
func (result geniusResult) compliant() bool {
	spec := util.UniqueFields(fmt.Sprintf("%s %s", result.Artist.Name, result.Title))
	return result.URL != "" &&
		util.Contains(spec, strings.Split(util.UniqueFields(result.track.Artists[0]), " ")...) &&
		util.Contains(spec, strings.Split(util.UniqueFields(result.track.Song()), " ")...)
}

// score goes from 0 to 100 and it's built on the accuracy percentage
// for result artist+title compared to the given track
func (result geniusResult) score() int {
	distance := int(math.Min(
		float64(levenshtein.ComputeDistance(
			util.UniqueFields(result.query),
			util.UniqueFields(fmt.Sprintf("%s %s", result.Artist.Name, result.Title)),
		)),
		50.0,
	))
	// return the inverse of the proportion of the distance
	// on a percentage scale to 50
	return 100 - (distance * 100 / 50)
}
