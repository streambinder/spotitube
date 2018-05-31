package track

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	spttb_system "system"

	"github.com/PuerkitoBio/goquery"
	"github.com/mozillazg/go-unidecode"
)

func searchLyricsGenius(track *Track) (string, error) {
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
		return "", errors.New("Unable to compile lyrics request: " + lyricsError.Error())
	}
	lyricsResponse, lyricsError := lyricsClient.Do(lyricsRequest)
	if lyricsError != nil {
		return "", errors.New("Unable to read response from lyrics request: " + lyricsError.Error())
	}
	lyricsResponseBody, lyricsError := ioutil.ReadAll(lyricsResponse.Body)
	if lyricsError != nil {
		return "", errors.New("Unable to get response body: " + lyricsError.Error())
	}
	lyricsData := LyricsAPIEntry{}
	lyricsError = json.Unmarshal(lyricsResponseBody, &lyricsData)
	if lyricsError != nil {
		return "", errors.New("Unable to parse json from response body: " + lyricsError.Error())
	}

	return strings.TrimSpace(unidecode.Unidecode(lyricsData.Lyrics)), nil
}
