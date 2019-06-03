package youtube

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"../track"

	"github.com/PuerkitoBio/goquery"
	"github.com/agnivade/levenshtein"
	"github.com/bradfitz/slice"
	"github.com/gosimple/slug"
)

// Tracks : Track array
type Tracks []Track

// Track : single YouTube search result struct
type Track struct {
	Track         *track.Track
	ID            string
	URL           string
	Title         string
	User          string
	Duration      int
	AffinityScore int
}

const (
	// YouTubeVideoPrefix : YouTube video prefix
	YouTubeVideoPrefix = "https://www.youtube.com"
	// YouTubeQueryURL : YouTube query URL
	YouTubeQueryURL = YouTubeVideoPrefix + "/results"
	// YouTubeQueryPattern : YouTube query URL parseable with *printf functions
	YouTubeQueryPattern = YouTubeQueryURL + "?q=%s"
	// YouTubeHTMLVideoSelector : YouTube entry video selector
	YouTubeHTMLVideoSelector = ".yt-uix-tile-link"
	// YouTubeHTMLDescSelector : YouTube entry description selector
	YouTubeHTMLDescSelector = ".yt-lockup-byline"
	// YouTubeHTMLDurationSelector : YouTube entry duration selector
	YouTubeHTMLDurationSelector = ".accessible-description"
	// YouTubeDurationTolerance : max video duration difference tolerance
	YouTubeDurationTolerance = 20 // second(s)
)

// QueryTracks : initialize a Tracks object by searching for Track results
func QueryTracks(track *track.Track) (Tracks, error) {
	var (
		doc         *goquery.Document
		queryString = fmt.Sprintf(YouTubeQueryPattern,
			strings.Replace(track.SearchPattern, " ", "+", -1))
	)
	request, _ := http.NewRequest("GET", queryString, nil)
	request.Header.Add("Accept-Language", "en")
	response, err := http.DefaultClient.Do(request)
	if err == nil {
		doc, _ = goquery.NewDocumentFromResponse(response)
	} else {
		doc, err = goquery.NewDocument(queryString)
	}
	if err != nil {
		return Tracks{}, fmt.Errorf(fmt.Sprintf("Cannot retrieve doc from \"%s\": %s", queryString, err.Error()))
	}
	html, _ := doc.Html()
	if strings.Contains(strings.ToLower(html), "unusual traffic") {
		return Tracks{}, fmt.Errorf("YouTube busted you: you'd better wait few minutes before retrying firing thousands video requests")
	}

	tracks, err := pullTracksFromDoc(*track, doc)
	if err != nil {
		return Tracks{}, err
	}

	tracks = tracks.evaluateScores()
	slice.Sort(tracks[:], func(i, j int) bool {
		var iPlus, jPlus int
		if tracks[i].AffinityScore == tracks[j].AffinityScore {
			if err := tracks[i].Track.Seems(fmt.Sprintf("%s %s", tracks[i].User, tracks[i].Title)); err == nil {
				iPlus = 1
			}
			if err := tracks[j].Track.Seems(fmt.Sprintf("%s %s", tracks[j].User, tracks[j].Title)); err == nil {
				jPlus = 1
			}
		}
		return tracks[i].AffinityScore+iPlus > tracks[j].AffinityScore+jPlus
	})
	return tracks, nil
}

// Match : return nil error if YouTube Track result object is matching with input Track object
func (youtube_track Track) Match(track track.Track) error {
	if int(math.Abs(float64(track.Duration-youtube_track.Duration))) > YouTubeDurationTolerance {
		return fmt.Errorf(fmt.Sprintf("The duration difference is excessive: | %d - %d | = %d (max tolerated: %d)",
			track.Duration, youtube_track.Duration, int(math.Abs(float64(track.Duration-youtube_track.Duration))), YouTubeDurationTolerance))
	}
	if strings.Contains(youtube_track.URL, "&list=") {
		return fmt.Errorf("Track is actually pointing to playlist")
	}
	if strings.Contains(youtube_track.URL, "/user/") {
		return fmt.Errorf("Track is actually pointing to user")
	}
	return track.Seems(fmt.Sprintf("%s %s", youtube_track.User, youtube_track.Title))
}

// Download : delegate youtube-dl call to download YouTube Track result
func (youtube_track Track) Download() error {
	var commandOut bytes.Buffer
	commandCmd := "youtube-dl"
	commandArgs := []string{"--output", fmt.Sprintf("%s.%s", youtube_track.Track.FilenameTemp, youtube_track.Track.FilenameExt[1:]), "--format", "bestaudio", "--extract-audio", "--audio-format", youtube_track.Track.FilenameExt[1:], "--audio-quality", "0", youtube_track.URL}
	commandObj := exec.Command(commandCmd, commandArgs...)
	commandObj.Stderr = &commandOut
	if commandErr := commandObj.Run(); commandErr != nil {
		return fmt.Errorf(fmt.Sprintf("Something went wrong while executing \"%s %s\":\n%s", commandCmd, strings.Join(commandArgs, " "), commandOut.String()))
	}
	return nil
}

// IDFromURL : extract YouTube entry ID from input URL
func IDFromURL(url string) string {
	var idPart string
	if strings.Contains(strings.ToLower(url), "youtu.be/") {
		idPart = strings.Split(url, "youtu.be/")[1]
	} else {
		idPart = strings.Split(url, "watch?v=")[1]
	}
	if strings.Contains(idPart, "?") {
		idPart = strings.Split(idPart, "?")[0]
	}
	if strings.Contains(idPart, "&list") {
		idPart = strings.Split(idPart, "&list")[0]
	}
	return idPart
}

// ValidateURL : return nil error if input URL is a valid YouTube URL
func ValidateURL(url string) error {
	if !strings.Contains(strings.ToLower(url), "youtu.be/") &&
		!strings.Contains(strings.ToLower(url), "watch?v=") {
		return fmt.Errorf(fmt.Sprintf("URL %s doesn't seem to be pointing to any YouTube video.", url))
	}
	return nil
}

func pullTracksFromDoc(track track.Track, document *goquery.Document) (Tracks, error) {
	var (
		tracks            = []Track{}
		selection         = document.Find(YouTubeHTMLVideoSelector)
		selectionDesc     = document.Find(YouTubeHTMLDescSelector)
		selectionDuration = document.Find(YouTubeHTMLDurationSelector)
		selectionPointer  int
		selectionError    error
	)
	for selectionPointer+1 < len(selection.Nodes) {
		selectionPointer++

		item := selection.Eq(selectionPointer)
		itemHref, itemHrefOk := item.Attr("href")
		itemTitle, itemTitleOk := item.Attr("title")
		itemUser, _ := "UNKNOWN", false
		itemLength, itemLengthOk := 0, false
		if selectionPointer < len(selectionDesc.Nodes) {
			itemDesc := selectionDesc.Eq(selectionPointer)
			itemUser = strings.TrimSpace(itemDesc.Find("a").Text())
			// itemUserOk = true
		}
		if selectionPointer < len(selectionDuration.Nodes) {
			var itemLengthMin, itemLengthSec int
			itemDuration := selectionDuration.Eq(selectionPointer)
			itemLengthSectr := strings.TrimSpace(itemDuration.Text())
			if strings.Contains(itemLengthSectr, ": ") {
				itemLengthSectr = strings.Split(itemLengthSectr, ": ")[1]
				itemLengthMin, selectionError = strconv.Atoi(strings.Split(itemLengthSectr, ":")[0])
				if selectionError == nil {
					itemLengthSec, selectionError = strconv.Atoi(strings.Split(itemLengthSectr, ":")[1][:2])
					if selectionError == nil {
						itemLength = itemLengthMin*60 + itemLengthSec
						itemLengthOk = true
					}
				}
			}
		}
		if itemHrefOk && itemTitleOk && itemLengthOk &&
			(strings.Contains(strings.ToLower(itemHref), "youtu.be") || !strings.Contains(strings.ToLower(itemHref), "&list=")) &&
			(strings.Contains(strings.ToLower(itemHref), "youtu.be") || strings.Contains(strings.ToLower(itemHref), "watch?v=")) {
			tracks = append(tracks, Track{
				Track:    &track,
				ID:       IDFromURL(YouTubeVideoPrefix + itemHref),
				URL:      YouTubeVideoPrefix + itemHref,
				Title:    itemTitle,
				User:     itemUser,
				Duration: itemLength,
			})
		}
	}

	return tracks, nil
}

func (tracks Tracks) evaluateScores() Tracks {
	var evaluatedTracks Tracks
	for _, t := range tracks {
		if math.Abs(float64(t.Track.Duration-t.Duration)) <= float64(YouTubeDurationTolerance/2) {
			t.AffinityScore += 20
		} else if math.Abs(float64(t.Track.Duration-t.Duration)) <= float64(YouTubeDurationTolerance) {
			t.AffinityScore += 10
		}
		if err := t.Track.SeemsByWordMatch(fmt.Sprintf("%s %s", t.User, t.Title)); err == nil {
			t.AffinityScore += 10
		}
		if strings.Contains(slug.Make(t.User), slug.Make(t.Track.Artist)) {
			t.AffinityScore += 10
		}
		if track.SeemsType(t.Title, t.Track.SongType) {
			t.AffinityScore += 10
		}
		levenshteinDistance := levenshtein.ComputeDistance(t.Track.SearchPattern, fmt.Sprintf("%s %s", t.User, t.Title))
		t.AffinityScore -= levenshteinDistance
		evaluatedTracks = append(evaluatedTracks, t)
	}
	return evaluatedTracks
}
