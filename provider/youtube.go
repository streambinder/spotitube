package provider

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/agnivade/levenshtein"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
	"github.com/tidwall/gjson"
)

type youTube struct {
	Provider
}

type youTubeResult struct {
	track  *entity.Track
	query  string
	id     string
	title  string
	owner  string
	views  int
	length int
}

const (
	titleScoreMultiplier    = 1.5
	durationScoreMultiplier = 2.0
	viewsScoreMultiplier    = 0.0000005
	keywordsMatchScore      = 100
	skimThreshold           = -100
)

func init() {
	providers = append(providers, youTube{})
}

func (provider youTube) search(track *entity.Track) ([]*Match, error) {
	searchKeys := url.Values{
		"search_query": append([]string{track.Title}, track.Artists...),
	}
	response, err := http.Get("https://www.youtube.com/results?" + searchKeys.Encode())
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, errors.New("cannot fetch results on youtube: " + response.Status)
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, err
	}

	resultJSON := strings.Join(document.Find("script").Map(func(i int, selection *goquery.Selection) string {
		if !strings.HasPrefix(strings.TrimPrefix(selection.Text(), " "), "var ytInitialData =") {
			return ""
		}
		return strings.TrimSpace(selection.Text()[19:])
	}), "")

	var matches []*Match
	gjson.Get(resultJSON, "contents.twoColumnSearchResultsRenderer.primaryContents.sectionListRenderer.contents.0.itemSectionRenderer.contents").ForEach(func(key, value gjson.Result) bool {
		match := youTubeResult{
			track: track,
			query: searchKeys.Get("search_query"),
			id:    gjson.Get(value.String(), "videoRenderer.videoId").String(),
			title: gjson.Get(value.String(), "videoRenderer.title.runs.0.text").String(),
			owner: gjson.Get(value.String(), "videoRenderer.ownerText.runs.0.text").String(),
			views: func(viewCount string) int {
				if viewCount == "" {
					return 0
				}
				return util.ErrWrap(0)(strconv.Atoi(strings.ReplaceAll(strings.Split(viewCount, " ")[0], ".", "")))
			}(gjson.Get(value.String(), "videoRenderer.viewCountText.simpleText").String()),
			length: func(length string) int {
				if length == "" {
					return 0
				}
				var (
					digits  = strings.Split(length, ":")
					minutes = util.ErrWrap(0)(strconv.Atoi(digits[0]))
					seconds = util.ErrWrap(0)(strconv.Atoi(digits[1]))
				)
				return minutes*60 + seconds
			}(gjson.Get(value.String(), "videoRenderer.lengthText.simpleText").String()),
		}

		if match.id == "" || match.title == "" || match.owner == "" {
			return true
		}

		if match.score() > skimThreshold {
			matches = append(matches, &Match{fmt.Sprintf("https://youtu.be/%s", match.id), match.score()})
		}

		return true
	})

	return matches, nil
}

func (result youTubeResult) score() int {
	var (
		queryDistinct  = util.UniqueFields(result.query)
		resultDistinct = util.UniqueFields(fmt.Sprintf("%s %s", result.owner, result.title))
		titleScore     = int(float64(levenshtein.ComputeDistance(queryDistinct, resultDistinct)) * titleScoreMultiplier)
		durationScore  = int(math.Abs(float64(result.length)-float64(result.track.Duration)) * durationScoreMultiplier)
		viewsScore     = int(float64(result.views) * viewsScoreMultiplier)
		score          = viewsScore - titleScore - durationScore
	)

	if strings.Contains(resultDistinct, util.Flatten(result.track.Artists[0])) &&
		strings.Contains(resultDistinct, util.Flatten(result.track.Title)) {
		score += keywordsMatchScore
	}

	return score
}
