package lyrics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/rainycape/unidecode"
	"github.com/streambinder/spotitube/src/system"
)

const (
	ovhAPI = "https://api.lyrics.ovh/v1/%s/%s"
)

type ovhAPIEntry struct {
	Lyrics string `json:"lyrics"`
}

// OVHProvider is the provider implementation which uses as source
// lyrics.ovh lyrics platform
type OVHProvider struct {
	Provider
}

// Name returns a human readable name for the provider
func (p OVHProvider) Name() string {
	return "lyrics.ovh"
}

// Query returns a lyrics text for give title and artist
func (p OVHProvider) Query(title, artist string) (string, error) {
	encURL, err := url.Parse(fmt.Sprintf(ovhAPI, url.QueryEscape(artist), url.QueryEscape(title)))
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, encURL.String(), nil)
	if err != nil {
		return "", err
	}

	res, err := system.Client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	entry := new(ovhAPIEntry)
	if err := json.Unmarshal(body, entry); err != nil {
		return "", err
	}

	return strings.TrimSpace(unidecode.Unidecode(entry.Lyrics)), nil
}
