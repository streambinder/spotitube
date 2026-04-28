package provider

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/streambinder/spotitube/entity"
)

const (
	qobuzAPIBase      = "https://www.qobuz.com/api.json/0.2"
	qobuzOpenShellURL = "https://open.qobuz.com/track/1"
	qobuzQualityMP3   = "5"
)

var (
	qobuzProxies = []string{
		"https://dab.yeet.su/api/stream?trackId=%s&quality=" + qobuzQualityMP3,
		"https://dabmusic.xyz/api/stream?trackId=%s&quality=" + qobuzQualityMP3,
	}
	qobuzBundleScriptPattern = regexp.MustCompile(`<script[^>]+src="([^"]+/js/main\.js|/resources/[^"]+/js/main\.js)"`)
	qobuzCredentialsPattern  = regexp.MustCompile(`app_id:"(?P<id>\d{9})",app_secret:"(?P<secret>[a-f0-9]{32})"`)

	qobuzCredMu       sync.Mutex
	qobuzCachedID     string
	qobuzCachedSecret string
)

type qobuz struct{}

func init() {
	providers = append(providers, qobuz{})
}

func (qobuz) search(track *entity.Track) ([]*Match, error) {
	trackID, err := qobuzSearchTrack(track)
	if err != nil || trackID == 0 {
		return nil, nil
	}

	cdnURL, err := qobuzCDNURL(strconv.FormatInt(trackID, 10))
	if err != nil {
		return nil, nil
	}

	return []*Match{{URL: cdnURL, Score: 100}}, nil
}

func qobuzCredentials() (string, string, error) {
	qobuzCredMu.Lock()
	defer qobuzCredMu.Unlock()

	if qobuzCachedID != "" && qobuzCachedSecret != "" {
		return qobuzCachedID, qobuzCachedSecret, nil
	}

	req, err := http.NewRequest(http.MethodGet, qobuzOpenShellURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("qobuz: shell returned %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	scriptMatch := qobuzBundleScriptPattern.FindSubmatch(body)
	if scriptMatch == nil {
		return "", "", fmt.Errorf("qobuz: bundle script not found in shell")
	}

	bundleURL := string(scriptMatch[1])
	if strings.HasPrefix(bundleURL, "/") {
		bundleURL = "https://open.qobuz.com" + bundleURL
	}

	bundleReq, err := http.NewRequest(http.MethodGet, bundleURL, nil)
	if err != nil {
		return "", "", err
	}
	bundleReq.Header.Set("User-Agent", "Mozilla/5.0")

	bundleResp, err := http.DefaultClient.Do(bundleReq)
	if err != nil {
		return "", "", err
	}
	defer bundleResp.Body.Close()

	if bundleResp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("qobuz: bundle returned %s", bundleResp.Status)
	}

	bundle, err := io.ReadAll(bundleResp.Body)
	if err != nil {
		return "", "", err
	}

	credMatch := qobuzCredentialsPattern.FindSubmatch(bundle)
	if credMatch == nil {
		return "", "", fmt.Errorf("qobuz: credentials not found in bundle")
	}

	qobuzCachedID = string(credMatch[1])
	qobuzCachedSecret = string(credMatch[2])

	return qobuzCachedID, qobuzCachedSecret, nil
}

func qobuzSearchTrack(track *entity.Track) (int64, error) {
	appID, appSecret, err := qobuzCredentials()
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf("%s %s", track.Song(), track.Artists[0])
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	// qobuz API signature is vestigial for track/search — not validated server-side;
	// use fnv128 to avoid importing crypto/md5
	h := fnv.New128()
	h.Write([]byte("tracksearch" + "query" + query + "limit" + "1" + ts + appSecret))
	sig := fmt.Sprintf("%x", h.Sum(nil))

	params := url.Values{
		"query":       {query},
		"limit":       {"1"},
		"app_id":      {appID},
		"request_ts":  {ts},
		"request_sig": {sig},
	}

	req, err := http.NewRequest(http.MethodGet, qobuzAPIBase+"/track/search?"+params.Encode(), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("X-App-Id", appID)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, nil
	}

	var payload struct {
		Tracks struct {
			Items []struct {
				ID        int64 `json:"id"`
				Performer struct {
					Name string `json:"name"`
				} `json:"performer"`
			} `json:"items"`
		} `json:"tracks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}

	items := payload.Tracks.Items
	if len(items) == 0 {
		return 0, nil
	}

	return items[0].ID, nil
}

func qobuzCDNURL(trackID string) (string, error) {
	for _, proxy := range qobuzProxies {
		resp, err := http.Get(fmt.Sprintf(proxy, trackID)) // nolint
		if err != nil {
			continue
		}

		var payload struct {
			URL string `json:"url"`
		}
		err = json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()

		if err == nil && payload.URL != "" {
			return payload.URL, nil
		}
	}

	return "", fmt.Errorf("qobuz: all proxies failed for track %s", trackID)
}
