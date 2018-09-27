package youtube

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"strings"

	spttb_track "track"

	"github.com/PuerkitoBio/goquery"
	"github.com/bradfitz/slice"
)

// QueryTracks : initialize a Tracks object by searching for Track results
func QueryTracks(track *spttb_track.Track) (Tracks, error) {
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
		return Tracks{}, fmt.Errorf("YouTube busted you: you'd better wait few minutes before retrying firing thousands video requests.")
	}

	tracks, err := pullTracksFromDoc(*track, doc)
	if err != nil {
		return Tracks{}, err
	} else {
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
}

// Match : return nil error if YouTube Track result object is matching with input Track object
func (youtube_track Track) Match(track spttb_track.Track) error {
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
