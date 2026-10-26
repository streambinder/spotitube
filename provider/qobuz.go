package provider

import (
	"crypto/md5" //nolint:gosec // md5 is the signature scheme mandated by the qobuz API, not a security control
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/streambinder/pango/entity"
)

const (
	qobuzAPIBase = "https://www.qobuz.com/api.json/0.2"
	// two apps, two jobs: catalog search runs under the
	// open.qobuz.com embed app, which answers unsigned; stream
	// resolution runs under the play.qobuz.com web player production
	// app, the one holding streaming rights and the one the user's
	// token is issued to (the embed app resolves samples only, and
	// the player app refuses unsigned catalog calls)
	qobuzOpenShellURL = "https://open.qobuz.com/track/1"
	qobuzPlayShellURL = "https://play.qobuz.com/login"
	qobuzQualityMP3   = "5"
	// environment variable carrying the user's own qobuz
	// user_auth_token, taken from a logged-in play.qobuz.com session;
	// without it the API only serves 30-second samples
	qobuzTokenEnv = "QOBUZ_TOKEN" // #nosec G101 -- the name of the environment variable, not a credential
)

var (
	qobuzBundleScriptPattern     = regexp.MustCompile(`<script[^>]+src="(/resources/[^"]+\.js)"`)
	qobuzOpenBundleScriptPattern = regexp.MustCompile(`<script[^>]+src="([^"]+/js/main\.js|/resources/[^"]+/js/main\.js)"`)
	qobuzOpenCredentialsPattern  = regexp.MustCompile(`app_id:"(?P<id>\d{9})",app_secret:"(?P<secret>[a-f0-9]{32})"`)
	// the web player bundle carries one api config per environment;
	// the production entry is the pair the player itself signs with,
	// with a generic pair as fallback if the labels move
	qobuzCredentialsPatterns = []*regexp.Regexp{
		regexp.MustCompile(`production:\{api:\{appId:"(?P<id>\d+)",appSecret:"(?P<secret>[a-f0-9]{32})"\}`),
		regexp.MustCompile(`appId:"(?P<id>\d+)",appSecret:"(?P<secret>[a-f0-9]{32})"`),
	}

	qobuzHTTPClient = &http.Client{Timeout: 10 * time.Second}

	qobuzCredMu        sync.Mutex
	qobuzCachedID      string
	qobuzCachedSecrets []string
	qobuzFileSecret    string

	qobuzSearchCachedID     string
	qobuzSearchCachedSecret string

	// stream url cache: trackID → url, avoids re-resolving the same
	// track across playlists within a single sync run
	qobuzURLCache sync.Map
)

type qobuz struct{}

func init() {
	providers = append(providers, qobuz{})
}

func (qobuz) name() string {
	return "qobuz"
}

func (qobuz) search(track *entity.Track) ([]*Match, error) {
	trackID, err := qobuzSearchTrack(track)
	if err != nil {
		return nil, err
	}
	if trackID == 0 {
		return nil, nil
	}

	streamURL, err := qobuzFileURL(strconv.FormatInt(trackID, 10))
	if err != nil {
		return nil, err
	}

	return []*Match{{URL: streamURL, Score: 100}}, nil
}

// qobuzFileCredentials scrapes the web player app ID and the
// candidate signing secrets for stream resolution from the
// play.qobuz.com bundle. The app secret printed in the bundle
// config does not sign: the signing secrets are derived from the
// bundle seed data (see qobuzDeriveSecrets), and the API accepts
// exactly one of the candidates, so callers try them in order.
func qobuzFileCredentials() (string, []string, error) {
	qobuzCredMu.Lock()
	defer qobuzCredMu.Unlock()

	if qobuzCachedID != "" && len(qobuzCachedSecrets) > 0 {
		return qobuzCachedID, qobuzCachedSecrets, nil
	}

	bundle, err := qobuzFetchBundle(qobuzPlayShellURL, qobuzBundleScriptPattern)
	if err != nil {
		return "", nil, err
	}

	var credMatch [][]byte
	for _, pattern := range qobuzCredentialsPatterns {
		if credMatch = pattern.FindSubmatch(bundle); credMatch != nil {
			break
		}
	}
	if credMatch == nil {
		return "", nil, fmt.Errorf("credentials not found in bundle")
	}

	qobuzCachedID = string(credMatch[1])
	qobuzCachedSecrets = qobuzDeriveSecrets(bundle)
	if len(qobuzCachedSecrets) == 0 {
		qobuzCachedSecrets = []string{string(credMatch[2])}
	}

	return qobuzCachedID, qobuzCachedSecrets, nil
}

var qobuzSeedTimezonePattern = regexp.MustCompile(`[a-z]\.initialSeed\("(?P<seed>[\w=]+)",window\.utimezone\.(?P<timezone>[a-z]+)\)`)

// qobuzDeriveSecrets reconstructs the candidate signing secrets
// hidden in the web player bundle: each environment seed is joined
// with the info and extras fragments registered for its timezone,
// the last 44 characters are dropped, and the rest is
// base64-decoded. The second timezone found takes priority, which
// mirrors the conditional order in the bundle.
func qobuzDeriveSecrets(bundle []byte) []string {
	type zone struct {
		name  string
		parts []string
	}
	var (
		zones  []*zone
		byName = map[string]*zone{}
	)
	for _, match := range qobuzSeedTimezonePattern.FindAllSubmatch(bundle, -1) {
		name := string(match[2])
		entry := byName[name]
		if entry == nil {
			entry = &zone{name: name}
			byName[name] = entry
			zones = append(zones, entry)
		}
		entry.parts = []string{string(match[1])}
	}
	if len(zones) == 0 {
		return nil
	}
	if len(zones) >= 2 {
		reordered := make([]*zone, 0, len(zones))
		reordered = append(reordered, zones[1], zones[0])
		zones = append(reordered, zones[2:]...)
	}

	capitalized := make([]string, len(zones))
	for i, entry := range zones {
		capitalized[i] = strings.ToUpper(entry.name[:1]) + entry.name[1:]
	}
	infoPattern := regexp.MustCompile(`name:"\w+/(?P<timezone>` + strings.Join(capitalized, "|") +
		`)",info:"(?P<info>[\w=]+)",extras:"(?P<extras>[\w=]+)"`)
	for _, match := range infoPattern.FindAllSubmatch(bundle, -1) {
		if entry := byName[strings.ToLower(string(match[1]))]; entry != nil {
			entry.parts = append(entry.parts, string(match[2]), string(match[3]))
		}
	}

	secrets := []string{}
	for _, entry := range zones {
		joined := strings.Join(entry.parts, "")
		if len(joined) <= 44 {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(joined[:len(joined)-44])
		if err != nil || len(decoded) == 0 {
			continue
		}
		secrets = append(secrets, string(decoded))
	}

	return secrets
}

// qobuzSearchCredentials scrapes the embed app credentials used for
// catalog search, from the open.qobuz.com bundle.
func qobuzSearchCredentials() (string, string, error) {
	qobuzCredMu.Lock()
	defer qobuzCredMu.Unlock()

	if qobuzSearchCachedID != "" && qobuzSearchCachedSecret != "" {
		return qobuzSearchCachedID, qobuzSearchCachedSecret, nil
	}

	bundle, err := qobuzFetchBundle(qobuzOpenShellURL, qobuzOpenBundleScriptPattern)
	if err != nil {
		return "", "", err
	}

	credMatch := qobuzOpenCredentialsPattern.FindSubmatch(bundle)
	if credMatch == nil {
		return "", "", fmt.Errorf("credentials not found in bundle")
	}

	qobuzSearchCachedID = string(credMatch[1])
	qobuzSearchCachedSecret = string(credMatch[2])

	return qobuzSearchCachedID, qobuzSearchCachedSecret, nil
}

// qobuzFetchBundle fetches a qobuz shell page and returns the
// JavaScript bundle the shell loads.
func qobuzFetchBundle(shellURL string, scriptPattern *regexp.Regexp) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, shellURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := qobuzHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("shell returned %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	scriptMatch := scriptPattern.FindSubmatch(body)
	if scriptMatch == nil {
		return nil, fmt.Errorf("bundle script not found in shell")
	}

	bundleURL := string(scriptMatch[1])
	if strings.HasPrefix(bundleURL, "/") {
		shell, parseErr := url.Parse(shellURL)
		if parseErr != nil {
			return nil, parseErr
		}
		bundleURL = shell.Scheme + "://" + shell.Host + bundleURL
	}

	bundleReq, err := http.NewRequest(http.MethodGet, bundleURL, nil)
	if err != nil {
		return nil, err
	}
	bundleReq.Header.Set("User-Agent", "Mozilla/5.0")

	bundleResp, err := qobuzHTTPClient.Do(bundleReq)
	if err != nil {
		return nil, err
	}
	defer bundleResp.Body.Close()

	if bundleResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bundle returned %s", bundleResp.Status)
	}

	return io.ReadAll(bundleResp.Body)
}

func qobuzSearchTrack(track *entity.Track) (int64, error) {
	appID, appSecret, err := qobuzSearchCredentials()
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf("%s %s", track.Song(), track.Artist())
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	// MD5 over the endpoint name, the arguments in key order, the
	// timestamp and the app secret, as for track/getFileUrl
	sigBytes := md5.Sum([]byte( //nolint:gosec // mandated by the qobuz API signature scheme
		"tracksearch" + "limit" + "1" + "query" + query + ts + appSecret))
	sig := fmt.Sprintf("%x", sigBytes)

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

	resp, err := qobuzHTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("search returned %s", resp.Status)
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

// qobuzFileURL resolves a track ID to a full stream URL through the
// official track/getFileUrl endpoint, signed with one of the
// derived app secrets and authenticated with the user's own token.
// A candidate secret whose signature the API rejects (400) falls
// through to the next candidate; the first secret the API accepts
// is remembered for the rest of the process. Results are cached
// per-process to avoid redundant lookups across playlists.
func qobuzFileURL(trackID string) (string, error) {
	if cached, ok := qobuzURLCache.Load(trackID); ok {
		return cached.(string), nil
	}

	token := os.Getenv(qobuzTokenEnv)
	if token == "" {
		return "", fmt.Errorf("%s is not set: without a user token the API only serves 30-second samples", qobuzTokenEnv)
	}

	appID, candidates, err := qobuzFileCredentials()
	if err != nil {
		return "", err
	}
	if qobuzFileSecret != "" {
		candidates = []string{qobuzFileSecret}
	}

	var lastErr error
	for _, secret := range candidates {
		streamURL, rejected, err := qobuzFileURLAttempt(trackID, appID, secret, token)
		if rejected {
			lastErr = err
			continue
		}
		if err != nil {
			return "", err
		}
		qobuzFileSecret = secret
		qobuzURLCache.Store(trackID, streamURL)
		return streamURL, nil
	}

	return "", lastErr
}

// qobuzFileURLAttempt performs one signed getFileUrl call. The
// rejected flag reports a 400 answer, which is how the API refuses
// a signature built with the wrong candidate secret.
func qobuzFileURLAttempt(trackID, appID, secret, token string) (string, bool, error) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sigBytes := md5.Sum([]byte( //nolint:gosec // mandated by the qobuz API signature scheme
		"trackgetFileUrl" + "format_id" + qobuzQualityMP3 + "intentstream" + "track_id" + trackID + ts + secret))
	sig := fmt.Sprintf("%x", sigBytes)

	params := url.Values{
		"track_id":    {trackID},
		"format_id":   {qobuzQualityMP3},
		"intent":      {"stream"},
		"app_id":      {appID},
		"request_ts":  {ts},
		"request_sig": {sig},
	}

	req, err := http.NewRequest(http.MethodGet, qobuzAPIBase+"/track/getFileUrl?"+params.Encode(), nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("X-App-Id", appID)
	req.Header.Set("X-User-Auth-Token", token)

	resp, err := qobuzHTTPClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		return "", true, fmt.Errorf("getFileUrl returned %s", resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("getFileUrl returned %s", resp.Status)
	}

	var payload struct {
		URL          string `json:"url"`
		Sample       bool   `json:"sample"`
		Restrictions []struct {
			Code string `json:"code"`
		} `json:"restrictions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", false, err
	}
	if payload.URL == "" {
		return "", false, fmt.Errorf("getFileUrl returned no URL for track %s", trackID)
	}
	if payload.Sample {
		codes := make([]string, 0, len(payload.Restrictions))
		for _, restriction := range payload.Restrictions {
			codes = append(codes, restriction.Code)
		}
		return "", false, fmt.Errorf("getFileUrl returned a 30-second sample for track %s (restrictions: %s): the token in %s is invalid or has no streaming entitlement",
			trackID, strings.Join(codes, ", "), qobuzTokenEnv)
	}

	return payload.URL, false, nil
}
