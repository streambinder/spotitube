package provider

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
)

type youTube struct {
	Provider
}

type youTubeInitialData struct {
	Contents struct {
		TwoColumnSearchResultsRenderer struct {
			PrimaryContents struct {
				SectionListRenderer struct {
					Contents []struct {
						ItemSectionRenderer struct {
							Contents []struct {
								VideoRenderer struct {
									VideoId string
									Title   struct {
										Runs []Run
									}
									OwnerText struct {
										Runs []Run
									}
									ViewCountText struct {
										SimpleText string
									}
									LengthText struct {
										SimpleText string
									}
									PublishedTimeText struct {
										SimpleText string
									}
									DetailedMetadataSnippets []DetailedMetadataSnippet
									OwnerBadges              []OwnerBadge
								}
							}
						}
					}
				}
			}
		}
	}
}

type DetailedMetadataSnippet struct {
	SnippetText SnippetText
}

type OwnerBadge struct {
	MetadataBadgeRenderer MetadataBadgeRenderer
}
type MetadataBadgeRenderer struct {
	Icon Icon
}

type Icon struct {
	IconType string
}

type SnippetText struct {
	Runs []Run
}

type Run struct {
	Text string
}

type youTubeResult struct {
	track                 *entity.Track
	query                 string
	id                    string
	title                 string
	owner                 string
	description           string
	views                 int
	length                int
	year                  int
	officialArtistChannel bool
	verifiedChannel       bool
}

func init() {
	providers = append(providers, youTube{})
}

func (provider youTube) search(track *entity.Track) ([]*Match, error) {
	query := track.Title
	for _, artist := range track.Artists {
		query = fmt.Sprintf("%s %s", query, artist)
	}

	response, err := http.Get("https://www.youtube.com/results?search_query=" + url.QueryEscape(query) + "&sp=EgIQAQ%253D%253D")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == 429 {
		util.SleepUntilRetry(response.Header)
		return provider.search(track)
	} else if response.StatusCode != 200 {
		return nil, errors.New("cannot fetch results on youtube: " + response.Status)
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, err
	}

	json := strings.Join(document.Find("script").Map(func(i int, selection *goquery.Selection) string {
		prefix := "var ytInitialData ="
		if !strings.HasPrefix(strings.TrimPrefix(selection.Text(), " "), prefix) {
			return ""
		}
		return strings.TrimSuffix(strings.TrimSpace(selection.Text()[len(prefix):]), ";")
	}), "")
	if json == "" {
		return []*Match{}, nil
	}

	var (
		matches []*Match
		data    youTubeInitialData
	)
	if err := jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal([]byte(json), &data); err != nil {
		return nil, err
	}
	for _, section := range data.Contents.TwoColumnSearchResultsRenderer.PrimaryContents.SectionListRenderer.Contents {
		for _, result := range section.ItemSectionRenderer.Contents {
			for run, title := range result.VideoRenderer.Title.Runs {
				match := youTubeResult{
					track: track,
					query: query,
					id:    result.VideoRenderer.VideoId,
					title: title.Text,
					owner: result.VideoRenderer.OwnerText.Runs[run].Text,
					description: util.First(result.VideoRenderer.DetailedMetadataSnippets, DetailedMetadataSnippet{
						SnippetText: SnippetText{Runs: []Run{{Text: ""}}},
					}).SnippetText.Runs[run].Text,
					views: func(viewCount string) int {
						if viewCount == "" {
							return 0
						}
						return util.ErrWrap(0)(strconv.Atoi(strings.Join(regexp.MustCompile("[0-9]+").FindAllString(viewCount, -1), "")))
					}(result.VideoRenderer.ViewCountText.SimpleText),
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
					}(result.VideoRenderer.LengthText.SimpleText),
					year: func(ago string) int {
						yearsAgo := 0
						if strings.Contains(ago, " year") {
							yearsAgo = util.ErrWrap(0)(strconv.Atoi(strings.Split(ago, " year")[0]))
						}
						return time.Now().Year() - yearsAgo
					}(result.VideoRenderer.PublishedTimeText.SimpleText),
					officialArtistChannel: func(iconType string) bool {
						return iconType == "OFFICIAL_ARTIST_BADGE"
					}(util.First(result.VideoRenderer.OwnerBadges, OwnerBadge{
						MetadataBadgeRenderer: MetadataBadgeRenderer{Icon: Icon{IconType: ""}},
					}).MetadataBadgeRenderer.Icon.IconType),
					verifiedChannel: func(iconType string) bool {
						return iconType == "CHECK_CIRCLE_THICK"
					}(util.First(result.VideoRenderer.OwnerBadges, OwnerBadge{
						MetadataBadgeRenderer: MetadataBadgeRenderer{Icon: Icon{IconType: ""}},
					}).MetadataBadgeRenderer.Icon.IconType),
				}
				if match.compliant(track) {
					matches = append(matches, &Match{fmt.Sprintf("https://youtu.be/%s", match.id), match.score(track)})
				}
			}
		}
	}

	return matches, nil
}

// compliance check works as a barrier before checking on the result score
// so to ensure that only the results that pass certain pre-checks get returned
func (result youTubeResult) compliant(track *entity.Track) bool {
	spec := util.UniqueFields(fmt.Sprintf("%s %s", result.owner, result.title))
	return result.id != "" && result.year >= track.Year &&
		util.Contains(spec, strings.Split(util.UniqueFields(track.Artists[0]), " ")...) &&
		util.Contains(spec, strings.Split(util.UniqueFields(track.Song()), " ")...)
}

// score goes from 0 to 100:
//
//	0–45% is derived from description score
//	0-25% is derived from duration score
//	0-15% is derived from views score
//	0-15% is derived from channel credibility score
func (result youTubeResult) score(track *entity.Track) int {
	var (
		descriptionScore = result.descriptionScore() * 45 / 100
		durationScore    = result.durationScore() * 25 / 100
		viewsScore       = result.viewsScore() * 15 / 100
		channelScore     = result.channelScore() * 15 / 100
	)
	return descriptionScore + durationScore + viewsScore + channelScore
}

// return a score for result description fields (i.e. owner, title, description)
func (result youTubeResult) descriptionScore() int {
	var (
		shortDescription = fmt.Sprintf("%s %s", result.title, result.owner)
		longDescription  = util.Flatten(fmt.Sprintf("%s %s", shortDescription, result.description))
	)
	distance := util.LevenshteinBoundedDistance(result.query, shortDescription)
	for _, word := range misleading {
		if util.Contains(longDescription, word) && !util.Contains(result.query, word) {
			distance += 10
		}
	}

	// return the inverse of the proportion of the distance
	// on a percentage scale to 50
	return 100 - int(math.Min(float64(distance), 50.0)*100/50)
}

// return a score for result duration
func (result youTubeResult) durationScore() int {
	distance := int(math.Min(math.Abs(float64(result.length)-float64(result.track.Duration)), 60.0))
	// return the inverse of the proportion of the distance
	// on a percentage scale to 60
	return 100 - (distance * 100 / 60)
}

// return a score for result's number of views
//
//	ie percentage of the number of digits of views on a scale to 11
//	(11 digits is for views of the order of 10.000.000.000, the highest reached on YouTube so far)
func (result youTubeResult) viewsScore() int {
	digits := len(strconv.Itoa(result.views))
	// boost results with more than a million views
	if digits > 6 {
		digits += 2
	}
	return int(math.Min(float64(digits), 11.0)) * 100 / 11
}

// return a score for result's channel according to its badges
//
//	0–70% is assigned if it's official
//	0-30% is assigned if it's verified
func (result youTubeResult) channelScore() int {
	return util.Ternary(result.officialArtistChannel, 70, 0) + util.Ternary(result.verifiedChannel, 30, 0)
}
