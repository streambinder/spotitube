package track

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	spttb_system "system"

	"github.com/PuerkitoBio/goquery"
	"github.com/kennygrant/sanitize"
	"github.com/mozillazg/go-unidecode"
)

func parseType(sequence string) int {
	for _, songType := range SongTypes {
		if SeemsType(sequence, songType) {
			return songType
		}
	}
	return SongTypeAlbum
}

func parseTitle(trackTitle string, trackFeaturings []string) (string, string) {
	var trackSong string

	trackTitle = strings.Split(trackTitle, " - ")[0]
	if strings.Contains(trackTitle, " live ") {
		trackTitle = strings.Split(trackTitle, " live ")[0]
	}
	trackTitle = strings.TrimSpace(trackTitle)
	if len(trackFeaturings) > 0 {
		var (
			featuringsAlreadyParsed bool
			featuringSymbols        = []string{"featuring", "feat", "ft", "with", "prod"}
		)
		for _, featuringValue := range trackFeaturings {
			for _, featuringSymbol := range featuringSymbols {
				if len(strings.Split(trackTitle, featuringSymbol)) > 1 &&
					strings.Contains(strings.ToLower(strings.Split(trackTitle, featuringSymbol)[1]), strings.ToLower(featuringValue)) {
					featuringsAlreadyParsed = true
				}
			}
		}
		if featuringsAlreadyParsed {
			for _, featuringSymbol := range featuringSymbols {
				for _, featuringSymbolCase := range []string{featuringSymbol, strings.Title(featuringSymbol)} {
					trackTitle = strings.Replace(trackTitle, featuringSymbolCase+". ", "ft. ", -1)
					trackTitle = strings.Replace(trackTitle, featuringSymbolCase+" ", "ft. ", -1)
				}
			}
		} else {
			if strings.Contains(trackTitle, "(") &&
				(strings.Contains(trackTitle, " vs. ") || strings.Contains(trackTitle, " vs ")) &&
				strings.Contains(trackTitle, ")") {
				trackTitle = strings.Split(trackTitle, " (")[0]
			}
			var trackFeaturingsInline string
			if len(trackFeaturings) > 1 {
				trackFeaturingsInline = "(ft. " + strings.Join(trackFeaturings[:len(trackFeaturings)-1], ", ") +
					" and " + trackFeaturings[len(trackFeaturings)-1] + ")"
			} else {
				trackFeaturingsInline = "(ft. " + trackFeaturings[0] + ")"
			}
			trackTitle = trackTitle + " " + trackFeaturingsInline
		}
		trackSong = strings.Split(trackTitle, " (ft. ")[0]
	} else {
		trackSong = trackTitle
	}

	return trackTitle, trackSong
}

func parseFilename(track Track) (string, string) {
	var (
		trackFilename     string
		trackFilenameTemp string
	)
	trackFilename = track.Artist + " - " + track.Title
	for _, symbol := range []string{"/", "\\", ".", "?", "<", ">", ":", "*"} {
		trackFilename = strings.Replace(trackFilename, symbol, "", -1)
	}

	// due to recent sanitize library changes, some umlauts changes
	// get performed instead of simple accents removal: while waiting for
	// https://github.com/kennygrant/sanitize/pull/23 request to be
	// merged (if ever), let's remove those symbols manually
	for _, umlaut := range [][]string{
		[]string{"Ö", "O"},
		[]string{"Ü", "U"},
		[]string{"ä", "a"},
		[]string{"ö", "o"},
		[]string{"ü", "u"}} {
		trackFilename = strings.Replace(trackFilename, umlaut[0], umlaut[1], -1)
	}

	trackFilename = strings.Replace(trackFilename, "  ", " ", -1)
	trackFilename = sanitize.Accents(trackFilename)
	trackFilename = strings.TrimSpace(trackFilename)
	trackFilenameTemp = sanitize.Name("." + trackFilename)

	return trackFilename, trackFilenameTemp
}

func searchLyricsGenius(track *Track) (string, error) {
	var geniusToken = os.Getenv("GENIUS_TOKEN")
	if len(geniusToken) == 0 {
		geniusToken = GeniusAccessToken
	}
	if len(GeniusAccessToken) == 0 {
		return "", fmt.Errorf("Cannot fetch lyrics from Genius without a valid token")
	}

	lyricsClient := http.Client{
		Timeout: time.Second * spttb_system.HTTPTimeout,
	}
	lyricsRequest, lyricsError := http.NewRequest(http.MethodGet,
		fmt.Sprintf(LyricsGeniusAPIURL, url.QueryEscape(track.Title), url.QueryEscape(track.Artist)), nil)
	if lyricsError != nil {
		return "", fmt.Errorf("Unable to compile Genius lyrics request: " + lyricsError.Error())
	}
	lyricsRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", GeniusAccessToken))

	lyricsResponse, lyricsError := lyricsClient.Do(lyricsRequest)
	if lyricsError != nil {
		return "", fmt.Errorf("Unable to read Genius lyrics response from lyrics request: " + lyricsError.Error())
	}

	lyricsResponseBody, lyricsError := ioutil.ReadAll(lyricsResponse.Body)
	if lyricsError != nil {
		return "", fmt.Errorf("Unable to get Genius lyrics response body: " + lyricsError.Error())
	}

	var result map[string]interface{}
	hitsUnmarshalErr := json.Unmarshal([]byte(lyricsResponseBody), &result)
	if hitsUnmarshalErr != nil {
		return "", fmt.Errorf("Unable to unmarshal Genius lyrics content into interface: %s", hitsUnmarshalErr.Error())
	}

	hits := result["response"].(map[string]interface{})["hits"].([]interface{})
	var lyricsURL string
	for _, value := range hits {
		valueResult := value.(map[string]interface{})["result"].(map[string]interface{})
		songTitle := strings.TrimSpace(valueResult["title"].(string))
		songArtist := strings.TrimSpace(valueResult["primary_artist"].(map[string]interface{})["name"].(string))

		songErr := track.Seems(fmt.Sprintf("%s %s", songTitle, songArtist))
		if songErr == nil {
			lyricsURL = strings.TrimSpace(valueResult["url"].(string))
			break
		}
	}

	if len(lyricsURL) == 0 {
		return "", fmt.Errorf("Genius lyrics not found")
	}

	doc, _ := goquery.NewDocument(lyricsURL)
	return strings.TrimSpace(unidecode.Unidecode(doc.Find(".lyrics").Eq(0).Text())), nil
}

func searchLyricsOvh(track *Track) (string, error) {
	type LyricsAPIEntry struct {
		Lyrics string `json:"lyrics"`
	}
	lyricsClient := http.Client{
		Timeout: time.Second * spttb_system.HTTPTimeout,
	}
	lyricsRequest, lyricsError := http.NewRequest(http.MethodGet,
		fmt.Sprintf(LyricsOVHAPIURL, url.QueryEscape(track.Artist), url.QueryEscape(track.Song)), nil)
	if lyricsError != nil {
		return "", fmt.Errorf("Unable to compile lyrics request: " + lyricsError.Error())
	}
	lyricsResponse, lyricsError := lyricsClient.Do(lyricsRequest)
	if lyricsError != nil {
		return "", fmt.Errorf("Unable to read response from lyrics request: " + lyricsError.Error())
	}
	lyricsResponseBody, lyricsError := ioutil.ReadAll(lyricsResponse.Body)
	if lyricsError != nil {
		return "", fmt.Errorf("Unable to get response body: " + lyricsError.Error())
	}
	lyricsData := LyricsAPIEntry{}
	lyricsError = json.Unmarshal(lyricsResponseBody, &lyricsData)
	if lyricsError != nil {
		return "", fmt.Errorf("Unable to parse json from response body: " + lyricsError.Error())
	}

	return strings.TrimSpace(unidecode.Unidecode(lyricsData.Lyrics)), nil
}
