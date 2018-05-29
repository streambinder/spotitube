package youtube

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	spttb_system "system"
	spttb_track "track"

	"github.com/PuerkitoBio/goquery"
)

// Tracks : simple iterator-like struct to easily loop over YouTube search results
type Tracks struct {
	Track             *spttb_track.Track
	Selection         *goquery.Selection
	SelectionDesc     *goquery.Selection
	SelectionDuration *goquery.Selection
	SelectionPointer  int
}

// Track : single YouTube search result struct
type Track struct {
	Track    *spttb_track.Track
	ID       string
	URL      string
	Title    string
	User     string
	Duration int
}

// QueryTracks : initialize a Tracks object by searching for Track results
func QueryTracks(track *spttb_track.Track) (*Tracks, error) {
	var (
		doc         *goquery.Document
		queryString = fmt.Sprintf(spttb_system.YouTubeQueryPattern,
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
		return &Tracks{}, fmt.Errorf(fmt.Sprintf("Cannot retrieve doc from \"%s\": %s", queryString, err.Error()))
	}
	html, _ := doc.Html()
	if strings.Contains(strings.ToLower(html), "unusual traffic") {
		return &Tracks{}, fmt.Errorf("YouTube busted you: you'd better wait few minutes before retrying firing thousands video requests.")
	}
	return &Tracks{
		Track:             track,
		Selection:         doc.Find(spttb_system.YouTubeHTMLVideoSelector),
		SelectionDesc:     doc.Find(spttb_system.YouTubeHTMLDescSelector),
		SelectionDuration: doc.Find(spttb_system.YouTubeHTMLDurationSelector),
		SelectionPointer:  0,
	}, nil
}

// HasNext : return True if current Tracks object still contains results
func (youtube_tracks *Tracks) HasNext() bool {
	return youtube_tracks.SelectionPointer+1 < len(youtube_tracks.Selection.Nodes)
}

// Next : get next Track object from Tracks
func (youtube_tracks *Tracks) Next() (*Track, error) {
	var err error
	if youtube_tracks.HasNext() {
		youtube_tracks.SelectionPointer++
		item := youtube_tracks.Selection.Eq(youtube_tracks.SelectionPointer)
		itemHref, itemHrefOk := item.Attr("href")
		itemTitle, itemTitleOk := item.Attr("title")
		itemUser, itemUserOk := "UNKNOWN", false
		itemLength, itemLengthOk := 0, false
		if youtube_tracks.SelectionPointer < len(youtube_tracks.SelectionDesc.Nodes) {
			itemDesc := youtube_tracks.SelectionDesc.Eq(youtube_tracks.SelectionPointer)
			itemUser = strings.TrimSpace(itemDesc.Find("a").Text())
			itemUserOk = true
		}
		if youtube_tracks.SelectionPointer < len(youtube_tracks.SelectionDuration.Nodes) {
			var itemLengthMin, itemLengthSec int
			itemDuration := youtube_tracks.SelectionDuration.Eq(youtube_tracks.SelectionPointer)
			itemLengthSectr := strings.TrimSpace(itemDuration.Text())
			if strings.Contains(itemLengthSectr, ": ") {
				itemLengthSectr = strings.Split(itemLengthSectr, ": ")[1]
				itemLengthMin, err = strconv.Atoi(strings.Split(itemLengthSectr, ":")[0])
				if err == nil {
					itemLengthSec, err = strconv.Atoi(strings.Split(itemLengthSectr, ":")[1][:2])
					if err == nil {
						itemLength = itemLengthMin*60 + itemLengthSec
						itemLengthOk = true
					}
				}
			}
		}
		if !(itemHrefOk && itemTitleOk && itemLengthOk) {
			return &Track{}, fmt.Errorf(fmt.Sprintf("Non-standard YouTube video entry structure: "+
				"url is %s, title is %s, user is %s, duration is %s.",
				strconv.FormatBool(itemHrefOk), strconv.FormatBool(itemTitleOk),
				strconv.FormatBool(itemUserOk), strconv.FormatBool(itemLengthOk)))
		} else if !strings.Contains(strings.ToLower(itemHref), "youtu.be") &&
			strings.Contains(strings.ToLower(itemHref), "&list=") {
			return &Track{}, fmt.Errorf(fmt.Sprintf("Playlist URL found: %s", itemHref))
		} else if !strings.Contains(strings.ToLower(itemHref), "youtu.be") &&
			!strings.Contains(strings.ToLower(itemHref), "watch?v=") {
			return &Track{}, fmt.Errorf(fmt.Sprintf("Advertising URL found: %s", itemHref))
		}

		return &Track{
			Track:    youtube_tracks.Track,
			ID:       IDFromURL(spttb_system.YouTubeVideoPrefix + itemHref),
			URL:      spttb_system.YouTubeVideoPrefix + itemHref,
			Title:    itemTitle,
			User:     itemUser,
			Duration: itemLength,
		}, nil
	}

	return &Track{}, fmt.Errorf("No more results left on page")
}

// Match : return nil error if YouTube Track result object is matching with input Track object
func (youtube_track Track) Match(track spttb_track.Track) error {
	if int(math.Abs(float64(track.Duration-youtube_track.Duration))) > spttb_system.YouTubeDurationTolerance {
		return fmt.Errorf(fmt.Sprintf("The duration difference is excessive: | %d - %d | = %d (max tolerated: %d)",
			track.Duration, youtube_track.Duration, int(math.Abs(float64(track.Duration-youtube_track.Duration))), spttb_system.YouTubeDurationTolerance))
	}
	if strings.Contains(youtube_track.URL, "&list=") || strings.Contains(youtube_track.URL, "/user/") {
		return fmt.Errorf("Track is actually pointing to playlist or user")
	}
	return track.Seems(youtube_track.Title)
}

// Download : delegate youtube-dl call to download YouTube Track result
func (youtube_track Track) Download() error {
	var commandOut bytes.Buffer
	commandCmd := "youtube-dl"
	commandArgs := []string{"--output", youtube_track.Track.FilenameTemp + ".%(ext)s", "--format", "bestaudio", "--extract-audio", "--audio-format", youtube_track.Track.FilenameExt[1:], "--audio-quality", "0", youtube_track.URL}
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
