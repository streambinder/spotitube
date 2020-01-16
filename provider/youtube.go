package provider

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bradfitz/slice"
	"github.com/streambinder/spotitube/shell"
	"github.com/streambinder/spotitube/track"
)

const (
	youTubeVideoPrefix          = "https://www.youtube.com"
	youTubeQueryURL             = youTubeVideoPrefix + "/results"
	youTubeQueryPattern         = youTubeQueryURL + "?q=%s"
	youTubeHTMLVideoSelector    = ".yt-uix-tile-link"
	youTubeHTMLDescSelector     = ".yt-lockup-byline"
	youTubeHTMLDurationSelector = ".accessible-description"
)

var (
	regURL = regexp.MustCompile(`(?m)(?:youtube\.com\/(?:[^\/]+\/.+\/|(?:v|e(?:mbed)?)\/|.*[?&]v=)|youtu\.be\/)([^"&?\/ ]{11})`)
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

// Query searches provider for entries related to track
func (p YouTubeProvider) Query(track *track.Track) ([]*Entry, error) {
	var queryString = fmt.Sprintf(youTubeQueryPattern, strings.Replace(track.Query(), " ", "+", -1))

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

// Match returns nil error if YouTube entry is matching with track
func (p YouTubeProvider) Match(entry *Entry, track *track.Track) error {
	if err := p.Support(entry.URL); err != nil {
		return err
	}

	if int(math.Abs(float64(track.Duration-entry.Duration))) > DurationTolerance {
		return fmt.Errorf("The duration delta too high")
	}

	return track.Seems(fmt.Sprintf("%s %s", entry.User, entry.Title))
}

// Download handles the youtube-dl call to download entry
func (p YouTubeProvider) Download(e *Entry, fname string) error {
	var (
		ext  = strings.Replace(filepath.Ext(fname), ".", "", -1)
		base = fname[0 : len(fname)-(len(ext)+1)]
	)

	return shell.YoutubeDL().Download(e.URL, base, ext)
}

// Support returns nil error if input URL is a valid YouTube URL
func (p YouTubeProvider) Support(url string) error {
	if regURL.FindAllString(url, -1) == nil {
		return fmt.Errorf(fmt.Sprintf("URL %s doesn't seem to be pointing to any YouTube video.", url))
	}

	return nil
}

// IDFromURL extracts YouTube entry ID from input URL
func IDFromURL(url string) string {
	for _, group := range regURL.FindAllStringSubmatch(url, -1) {
		if len(group) > 1 {
			return group[1]
		}
	}
	return ""
}

func pullTracksFromDoc(track track.Track, document *goquery.Document) ([]*Entry, error) {
	var (
		entries  = []*Entry{}
		elVideo  = document.Find(youTubeHTMLVideoSelector)
		elDesc   = document.Find(youTubeHTMLDescSelector)
		elLength = document.Find(youTubeHTMLDurationSelector)
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
				ID:       IDFromURL(youTubeVideoPrefix + itemHref),
				URL:      youTubeVideoPrefix + itemHref,
				Title:    itemTitle,
				User:     itemUser,
				Duration: itemLength,
			})
		}
	}

	return entries, nil
}
