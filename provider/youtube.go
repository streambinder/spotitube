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
	query  string
	id     string
	title  string
	owner  string
	views  int
	length int
}

func init() {
	providers = append(providers, youTube{})
}

func (provider youTube) search(track *entity.Track) ([]*Match, error) {
	query := track.Title
	for _, artist := range track.Artists {
		query = fmt.Sprintf("%s %s", query, artist)
	}

	response, err := http.Get("https://www.youtube.com/results?search_query=" + url.QueryEscape(query))
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
			query: query,
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

		if match.compliant(track) {
			matches = append(matches, &Match{fmt.Sprintf("https://youtu.be/%s", match.id), match.score(track)})
		}

		return true
	})

	return matches, nil
}

// compliance check works as a barrier before checking on the result score
// so to ensure that only the results that pass certain pre-checks get returned
func (result youTubeResult) compliant(track *entity.Track) bool {
	return strings.Contains(
		util.UniqueFields(result.query),
		util.Flatten(track.Artists[0]),
	) && strings.Contains(
		util.UniqueFields(fmt.Sprintf("%s %s", result.owner, result.title)),
		util.Flatten(track.Title),
	)
}

// score goes from 0 to 100:
//
//	0–50% is added depending on the Levenshtein distance between the query and result owner+title
//	0-30% is added depending on the distance between the duration of the track and the result
//	0-20% is added depending on the amount of views the result has
func (result youTubeResult) score(track *entity.Track) int {
	var (
		titleScore    = result.titleAccuracy() / 2
		durationScore = result.durationAccuracy(track.Duration) * 3 / 10
		viewsScore    = result.viewsAccuracy() / 5
	)
	return titleScore + durationScore + viewsScore
}

// return an accuracy percentage for result owner+title
func (result youTubeResult) titleAccuracy() int {
	distance := int(math.Min(
		float64(levenshtein.ComputeDistance(
			util.UniqueFields(result.query),
			util.UniqueFields(fmt.Sprintf("%s %s", result.owner, result.title)),
		)),
		50.0,
	))
	// return the inverse of the proportion of the distance
	// on a percentage scale to 50
	return 100 - (distance * 100 / 50)
}

// return an accuracy percentage for result duration
func (result youTubeResult) durationAccuracy(duration int) int {
	distance := int(math.Min(math.Abs(float64(result.length)-float64(duration)), 60.0))
	// return the inverse of the proportion of the distance
	// on a percentage scale to 60
	return 100 - (distance * 100 / 60)
}

// return an accuracy percentage for number of views
//
//	ie percentage of the number of digits of views on a scale to 11
//	(11 digits is for views of the order of 10.000.000.000, the highest reached on YouTube so far)
func (result youTubeResult) viewsAccuracy() int {
	return int(math.Min(float64(len(strconv.Itoa(result.views))), 11.0)) * 100 / 11
}
