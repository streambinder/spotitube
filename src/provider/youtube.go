package provider

import (
	"bytes"
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bradfitz/slice"
	"github.com/streambinder/spotitube/src/track"
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

	regexURL = `(?m)(?:youtube\.com\/(?:[^\/]+\/.+\/|(?:v|e(?:mbed)?)\/|.*[?&]v=)|youtu\.be\/)([^"&?\/ ]{11})`
)

// YouTubeProvider is the provider implementation which uses as source
// YouTube videos.
type YouTubeProvider struct {
	Provider
	Scorer
}

// Name returns a human readable name for the provider
func (p YouTubeProvider) Name() string {
	return "YouTube"
}

// Query : query provider for entries related to track
func (p YouTubeProvider) Query(track *track.Track) ([]*Entry, error) {
	var queryString = fmt.Sprintf(YouTubeQueryPattern, strings.Replace(track.Query(), " ", "+", -1))

	d, err := goquery.NewDocument(queryString)
	if err != nil {
		return []*Entry{}, fmt.Errorf(fmt.Sprintf("Cannot retrieve doc from \"%s\": %s", queryString, err.Error()))
	}

	dContent, _ := d.Html()
	if strings.Contains(strings.ToLower(dContent), "unusual traffic") {
		return []*Entry{}, fmt.Errorf("YouTube busted you: you'd better wait few minutes before retrying firing thousands video requests")
	}

	entries, err := pullTracksFromDoc(*track, d)
	if err != nil {
		return []*Entry{}, err
	}

	slice.Sort(entries[:], func(i, j int) bool { return p.Score(entries[i], track) > p.Score(entries[j], track) })
	return entries, nil
}

// Match : return nil error if YouTube entry is matching with track
func (p YouTubeProvider) Match(e *Entry, t *track.Track) error {
	if int(math.Abs(float64(t.Duration-e.Duration))) > DurationTolerance {
		return fmt.Errorf(fmt.Sprintf("The duration difference is excessive: | %d - %d | = %d (max tolerated: %d)",
			t.Duration, e.Duration, int(math.Abs(float64(t.Duration-e.Duration))), DurationTolerance))
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
	var (
		ext = strings.Replace(filepath.Ext(fname), ".", "", -1)
		base = fname[0:len(fname)-(len(ext) + 1)]
	)

	cStderr := new(bytes.Buffer)
	c := exec.Command("youtube-dl", []string{
		"--format", "bestaudio", "--extract-audio",
		"--audio-format", ext,
		"--audio-quality", "0",
		"--output", base + ".%(ext)s", e.URL}...)
	c.Stderr = cStderr

	if cErr := c.Run(); cErr != nil {
		return fmt.Errorf(fmt.Sprintf("Unable to download from YouTube: \n%s", cStderr.String()))
	}

	return nil
}

// ValidateURL : return nil error if input URL is a valid YouTube URL
func (p YouTubeProvider) ValidateURL(url string) error {
	re := regexp.MustCompile(regexURL)
	if re.FindAllString(url, -1) == nil {
		return fmt.Errorf(fmt.Sprintf("URL %s doesn't seem to be pointing to any YouTube video.", url))
	}

	return nil
}

// IDFromURL : extract YouTube entry ID from input URL
func IDFromURL(url string) string {
	var id string

	if strings.Contains(strings.ToLower(url), "youtu.be/") {
		id = strings.Split(url, "youtu.be/")[1]
	} else {
		id = strings.Split(url, "watch?v=")[1]
	}

	if strings.Contains(id, "?") {
		id = strings.Split(id, "?")[0]
	}

	if strings.Contains(id, "&list") {
		id = strings.Split(id, "&list")[0]
	}

	return id
}

func pullTracksFromDoc(track track.Track, document *goquery.Document) ([]*Entry, error) {
	var (
		entries  = []*Entry{}
		elVideo  = document.Find(YouTubeHTMLVideoSelector)
		elDesc   = document.Find(YouTubeHTMLDescSelector)
		elLength = document.Find(YouTubeHTMLDurationSelector)
		elPtr    int
		elErr    error
	)
	for elPtr+1 < len(elVideo.Nodes) {
		elPtr++

		item := elVideo.Eq(elPtr)
		itemHref, itemHrefOk := item.Attr("href")
		itemTitle, itemTitleOk := item.Attr("title")
		itemUser, _ := "unknown", false
		itemLength, itemLengthOk := 0, false

		if elPtr < len(elDesc.Nodes) {
			itemDesc := elDesc.Eq(elPtr)
			itemUser = strings.TrimSpace(itemDesc.Find("a").Text())
			// itemUserOk = true
		}

		if elPtr < len(elLength.Nodes) {
			var itemLengthMin, itemLengthSec int
			itemDuration := elLength.Eq(elPtr)
			itemLengthSectr := strings.TrimSpace(itemDuration.Text())
			if strings.Contains(itemLengthSectr, ": ") {
				itemLengthSectr = strings.Split(itemLengthSectr, ": ")[1]
				itemLengthMin, elErr = strconv.Atoi(strings.Split(itemLengthSectr, ":")[0])
				if elErr == nil {
					itemLengthSec, elErr = strconv.Atoi(strings.Split(itemLengthSectr, ":")[1][:2])
					if elErr == nil {
						itemLength = itemLengthMin*60 + itemLengthSec
						itemLengthOk = true
					}
				}
			}
		}

		if itemHrefOk && itemTitleOk && itemLengthOk &&
			!strings.Contains(strings.ToLower(itemHref), "&list=") &&
			(strings.Contains(strings.ToLower(itemHref), "youtu.be") ||
				strings.Contains(strings.ToLower(itemHref), "watch?v=")) {
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
