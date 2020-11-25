package provider

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bradfitz/slice"
	"github.com/streambinder/spotitube/shell"
	"github.com/streambinder/spotitube/system"
	"github.com/streambinder/spotitube/track"
	"github.com/tidwall/gjson"
)

const (
	youTubeVideoPrefix       = "https://www.youtube.com"
	youTubeQueryURL          = youTubeVideoPrefix + "/results"
	youTubeQueryPattern      = youTubeQueryURL + "?q=%s"
	youTubeResultsLinePrefix = "var ytInitialData ="
	youTubeResultsLineSuffix = ";"
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

	entries, err := pullTracksFromDoc(*track, dContent)
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

	if int(math.Abs(float64(track.Duration-entry.Duration))) > durationDeltaTolerance {
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
		return fmt.Errorf(fmt.Sprintf("URL %s doesn't seem to be pointing to any YouTube video", url))
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

func pullTracksFromDoc(track track.Track, document string) ([]*Entry, error) {
	var entries = []*Entry{}
	if !strings.Contains(document, youTubeResultsLinePrefix) || !strings.Contains(document, youTubeResultsLineSuffix) {
		return entries, fmt.Errorf("No results found")
	}

	var json = strings.Split(strings.Split(document, youTubeResultsLinePrefix)[1], youTubeResultsLinePrefix)[0]
	gjson.Get(json, "contents.twoColumnSearchResultsRenderer.primaryContents.sectionListRenderer.contents.0.itemSectionRenderer.contents").ForEach(func(key, value gjson.Result) bool {
		e := &Entry{
			gjson.Get(value.String(), "videoRenderer.videoId").String(),
			"https://youtu.be/" + gjson.Get(value.String(), "videoRenderer.videoId").String(),
			gjson.Get(value.String(), "videoRenderer.title.runs.0.text").String(),
			gjson.Get(value.String(), "videoRenderer.ownerText.runs.0.text").String(),
			system.ColonDuration(gjson.Get(value.String(), "videoRenderer.lengthText.simpleText").String()),
		}
		if e.ID != "" && e.Title != "" && e.User != "" && e.Duration > 0 {
			entries = append(entries, e)
		}
		return true
	})

	return entries, nil
}
