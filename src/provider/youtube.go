package provider

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"../spotitube"
	"../track"

	"github.com/PuerkitoBio/goquery"
	"github.com/bradfitz/slice"
)

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

// YouTubeProvider is the provider implementation which uses as source
// YouTube videos.
type YouTubeProvider struct {
	Provider
	Scorer
}

// Query : query provider for entries related to track
func (p YouTubeProvider) Query(track *track.Track) ([]*Entry, error) {
	var (
		doc         *goquery.Document
		queryString = fmt.Sprintf(YouTubeQueryPattern,
			strings.Replace(track.Query(), " ", "+", -1))
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
		return []*Entry{}, fmt.Errorf(fmt.Sprintf("Cannot retrieve doc from \"%s\": %s", queryString, err.Error()))
	}
	html, _ := doc.Html()
	if strings.Contains(strings.ToLower(html), "unusual traffic") {
		return []*Entry{}, fmt.Errorf("YouTube busted you: you'd better wait few minutes before retrying firing thousands video requests")
	}

	entries, err := pullTracksFromDoc(*track, doc)
	if err != nil {
		return []*Entry{}, err
	}

	slice.Sort(entries[:], func(i, j int) bool { return p.Score(entries[i], track) > p.Score(entries[j], track) })
	return entries, nil
}

// Match : return nil error if YouTube entry is matching with track
func (p YouTubeProvider) Match(e *Entry, t *track.Track) error {
	if int(math.Abs(float64(t.Duration-e.Duration))) > YouTubeDurationTolerance {
		return fmt.Errorf(fmt.Sprintf("The duration difference is excessive: | %d - %d | = %d (max tolerated: %d)",
			t.Duration, e.Duration, int(math.Abs(float64(t.Duration-e.Duration))), YouTubeDurationTolerance))
	}
	if strings.Contains(e.URL, "&list=") {
		return fmt.Errorf("Track is actually pointing to playlist")
	}
	if strings.Contains(e.URL, "/user/") {
		return fmt.Errorf("Track is actually pointing to user")
	}
	return t.Seems(fmt.Sprintf("%s %s", e.User, e.Title))
}

// Download : delegate youtube-dl call to download entry
func (p YouTubeProvider) Download(e *Entry, fname string) error {
	var commandOut bytes.Buffer
	commandCmd := "youtube-dl"
	commandArgs := []string{"--format", "bestaudio", "--extract-audio", "--audio-format", spotitube.SongExtension, "--audio-quality", "0", "--output", strings.Replace(fname, fmt.Sprintf(".%s", spotitube.SongExtension), "", -1) + ".%(ext)s", e.URL}
	commandObj := exec.Command(commandCmd, commandArgs...)
	commandObj.Stderr = &commandOut
	if commandErr := commandObj.Run(); commandErr != nil {
		return fmt.Errorf(fmt.Sprintf("Something went wrong while executing \"%s %s\":\n%s", commandCmd, strings.Join(commandArgs, " "), commandOut.String()))
	}
	return nil
}

// ValidateURL : return nil error if input URL is a valid YouTube URL
func (p YouTubeProvider) ValidateURL(url string) error {
	if !strings.Contains(strings.ToLower(url), "youtu.be/") &&
		!strings.Contains(strings.ToLower(url), "watch?v=") {
		return fmt.Errorf(fmt.Sprintf("URL %s doesn't seem to be pointing to any YouTube video.", url))
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

func pullTracksFromDoc(track track.Track, document *goquery.Document) ([]*Entry, error) {
	var (
		entries           = []*Entry{}
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
			entries = append(entries, &Entry{
				ID:       IDFromURL(YouTubeVideoPrefix + itemHref),
				URL:      YouTubeVideoPrefix + itemHref,
				Title:    itemTitle,
				User:     itemUser,
				Duration: itemLength,
			})
		}
	}

	return entries, nil
}
