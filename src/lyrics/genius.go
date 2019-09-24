package lyrics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/rainycape/unidecode"
	"github.com/streambinder/spotitube/src/system"
)

const (
	geniusToken = ":GENIUS_TOKEN:"
	geniusAPI   = "https://api.genius.com/search?q=%s+%s"
)

// GeniusProvider is the provider implementation which uses as source
// Genius lyrics platform
type GeniusProvider struct {
	Provider
}

// Name returns a human readable name for the provider
func (p GeniusProvider) Name() string {
	return "Genius"
}

// Query returns a lyrics text for give title and artist
func (p GeniusProvider) Query(title, artist string) (string, error) {
	var token = geniusToken
	if envToken := os.Getenv("GENIUS_TOKEN"); len(envToken) != 64 {
		token = os.Getenv(envToken)
	}
	if len(token) != 64 {
		return "", fmt.Errorf("Cannot fetch lyrics from Genius without a valid token")
	}

	encURL, err := url.Parse(fmt.Sprintf(geniusAPI, url.QueryEscape(title), url.QueryEscape(artist)))
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, encURL.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	res, err := system.Client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return "", err
	}

	var (
		url  string
		hits = result["response"].(map[string]interface{})["hits"].([]interface{})
	)
	for _, hit := range hits {
		hitRes := hit.(map[string]interface{})["result"].(map[string]interface{})
		hitTitle := strings.TrimSpace(hitRes["title"].(string))
		hitArtist := strings.TrimSpace(hitRes["primary_artist"].(map[string]interface{})["name"].(string))

		if strings.Contains(hitTitle, title) && strings.Contains(hitArtist, artist) {
			url = strings.TrimSpace(hitRes["url"].(string))
			break
		}
	}

	if len(url) == 0 {
		return "", fmt.Errorf("Genius lyrics not found")
	}

	doc, _ := goquery.NewDocument(url)
	return strings.TrimSpace(unidecode.Unidecode(doc.Find(".lyrics").Eq(0).Text())), nil
}
