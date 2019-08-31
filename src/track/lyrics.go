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

	"../system"

	"github.com/PuerkitoBio/goquery"
	"github.com/rainycape/unidecode"
)

const (
	// GeniusAccessToken : Genius app access token
	GeniusAccessToken = ":GENIUS_TOKEN:"
	// LyricsGeniusAPIURL : lyrics Genius API URL
	LyricsGeniusAPIURL = "https://api.genius.com/search?q=%s+%s"
	// LyricsOVHAPIURL : lyrics OVH API URL
	LyricsOVHAPIURL = "https://api.lyrics.ovh/v1/%s/%s"
)

// Query : return string used to search song online
func (track Track) Query() string {
	return track.Basename()
}

// SearchLyrics : search Track lyrics, eventually throwing returning error
func (track *Track) SearchLyrics() error {
	var (
		lyrics    string
		lyricsErr error
	)
	lyrics, lyricsErr = searchLyricsGenius(track)
	if lyricsErr == nil {
		track.Lyrics = lyrics
		return nil
	}
	lyrics, lyricsErr = searchLyricsOvh(track)
	if lyricsErr == nil {
		track.Lyrics = lyrics
		return nil
	}
	return lyricsErr
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
		Timeout: time.Second * system.HTTPTimeout,
	}

	encodedURL, lyricsError := url.Parse(fmt.Sprintf(LyricsGeniusAPIURL, url.QueryEscape(track.Title), url.QueryEscape(track.Artist)))
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsRequest, lyricsError := http.NewRequest(http.MethodGet, encodedURL.String(), nil)
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", GeniusAccessToken))

	lyricsResponse, lyricsError := lyricsClient.Do(lyricsRequest)
	if lyricsError != nil {
		return "", lyricsError
	}

	lyricsResponseBody, lyricsError := ioutil.ReadAll(lyricsResponse.Body)
	if lyricsError != nil {
		return "", lyricsError
	}

	var result map[string]interface{}
	hitsUnmarshalErr := json.Unmarshal([]byte(lyricsResponseBody), &result)
	if hitsUnmarshalErr != nil {
		return "", hitsUnmarshalErr
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
		Timeout: time.Second * system.HTTPTimeout,
	}

	encodedURL, lyricsError := url.Parse(fmt.Sprintf(LyricsOVHAPIURL, url.QueryEscape(track.Artist), url.QueryEscape(track.Song)))
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsRequest, lyricsError := http.NewRequest(http.MethodGet, encodedURL.String(), nil)
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsResponse, lyricsError := lyricsClient.Do(lyricsRequest)
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsResponseBody, lyricsError := ioutil.ReadAll(lyricsResponse.Body)
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsData := LyricsAPIEntry{}
	lyricsError = json.Unmarshal(lyricsResponseBody, &lyricsData)
	if lyricsError != nil {
		return "", lyricsError
	}

	return strings.TrimSpace(unidecode.Unidecode(lyricsData.Lyrics)), nil
}
