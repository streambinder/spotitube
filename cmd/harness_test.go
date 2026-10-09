package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

// This file holds the shared fakes every command test runs against.
// Every network boundary of the application bottoms out in
// http.DefaultTransport (the Spotify API and OAuth2 endpoints through the
// spotify client, genius and lrclib through http.DefaultClient, youtube
// through http.Get, blob downloads through a client with a nil transport),
// and every external tool is invoked by bare name (ffmpeg, yt-dlp, the
// platform opener). Command tests therefore run the real code end to end,
// scripting only transports, PATH binaries and the filesystem.

const (
	tokenJSON = `{"access_token":"access","token_type":"Bearer",` +
		`"refresh_token":"refresh","expires_in":3600}`
	meJSON = `{"id":"alice","display_name":"Alice"}`
)

// trackJSON mirrors a full Spotify track payload for the given id.
// The canonical fixture track (id 123) is titled "Title"; any other id
// gets a distinct title, so multi-track scenarios do not collide on the
// final filename.
func trackJSON(id string) string {
	title := "Title"
	if id != "123" {
		title = "Title " + id
	}
	return fmt.Sprintf(`{"id":%q,"name":%q,"artists":[{"name":"Artist"}],`+
		`"duration_ms":180000,"track_number":1,`+
		`"album":{"name":"Album","release_date":"1970","images":[{"url":"http://ima.ge"}]}}`, id, title)
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

// installTransport swaps http.DefaultTransport for the given handler for
// the duration of the test.
func installTransport(t *testing.T, transport http.RoundTripper) {
	t.Helper()
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previous })
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		StatusCode:    status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

func apiError(status int, message string) (int, string) {
	return status, fmt.Sprintf(`{"error":{"status":%d,"message":%q}}`, status, message)
}

// spotifyRouter serves the Spotify Web API endpoints the commands use.
// Individual paths can be overridden per test through the overrides map,
// keyed by path prefix; everything else gets the standard fixtures.
// Longer prefixes win, so /v1/me/tracks beats /v1/me.
type spotifyRouter struct {
	overrides map[string]func(*http.Request) (int, string)
	prefixes  []string
}

func newSpotifyRouter(overrides map[string]func(*http.Request) (int, string)) spotifyRouter {
	prefixes := make([]string, 0, len(overrides))
	for prefix := range overrides {
		prefixes = append(prefixes, prefix)
	}
	sort.Slice(prefixes, func(i, j int) bool { return len(prefixes[i]) > len(prefixes[j]) })
	return spotifyRouter{overrides: overrides, prefixes: prefixes}
}

func (router spotifyRouter) RoundTrip(request *http.Request) (*http.Response, error) {
	for _, prefix := range router.prefixes {
		if strings.HasPrefix(request.URL.Path, prefix) {
			status, body := router.overrides[prefix](request)
			return jsonResponse(status, body), nil
		}
	}
	switch {
	case request.URL.Host == "accounts.spotify.com":
		return jsonResponse(200, tokenJSON), nil
	case request.URL.Path == "/v1/me":
		return jsonResponse(200, meJSON), nil
	case request.URL.Path == "/v1/me/tracks":
		if request.URL.Query().Get("offset") != "" {
			return jsonResponse(200, `{"items":[],"next":null,"total":1}`), nil
		}
		return jsonResponse(200, `{"items":[{"added_at":"2026-01-01T00:00:00Z","track":`+
			trackJSON("123")+`}],"next":null,"total":1}`), nil
	case strings.HasPrefix(request.URL.Path, "/v1/tracks/"):
		return jsonResponse(200, trackJSON(strings.TrimPrefix(request.URL.Path, "/v1/tracks/"))), nil
	case strings.HasPrefix(request.URL.Path, "/v1/albums/"):
		id := strings.TrimPrefix(request.URL.Path, "/v1/albums/")
		return jsonResponse(200, fmt.Sprintf(`{"id":%q,"name":"Album","artists":[{"name":"Artist"}],`+
			`"release_date":"1970","tracks":{"items":[%s],"next":null,"total":1}}`,
			id, simpleTrackJSON("123"))), nil
	case request.URL.Path == "/v1/me/playlists":
		return jsonResponse(200, `{"items":[{"id":"playlist123","name":"Mix"}],`+
			`"next":null,"total":1}`), nil
	case strings.HasPrefix(request.URL.Path, "/v1/playlists/"):
		id := strings.Trim(strings.TrimPrefix(request.URL.Path, "/v1/playlists/"), "/")
		if strings.HasSuffix(request.URL.Path, "/tracks") {
			return jsonResponse(200, `{"items":[{"track":`+trackJSON("123")+`}],"next":null,"total":1}`), nil
		}
		return jsonResponse(200, fmt.Sprintf(`{"id":%q,"name":"Mix",`+
			`"tracks":{"items":[{"track":%s}],"next":null,"total":1}}`, id, trackJSON("123"))), nil
	case request.URL.Path == "/v1/search":
		return jsonResponse(200, `{"tracks":{"items":[`+trackJSON("123")+`],"next":null,"total":1}}`), nil
	}
	return jsonResponse(404, `{"error":{"status":404,"message":"unknown"}}`), nil
}

func simpleTrackJSON(id string) string {
	return fmt.Sprintf(`{"id":%q,"name":"Title","artists":[{"name":"Artist"}],`+
		`"duration_ms":180000,"track_number":1}`, id)
}

// installSpotify installs the Spotify API router as the default transport.
func installSpotify(t *testing.T, overrides map[string]func(*http.Request) (int, string)) {
	t.Helper()
	installTransport(t, newSpotifyRouter(overrides))
}

// rawResponse builds a response with an explicit content type and body.
func rawResponse(status int, contentType string, body []byte) *http.Response {
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		StatusCode:    status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": []string{contentType}},
		Body:          io.NopCloser(strings.NewReader(string(body))),
		ContentLength: int64(len(body)),
	}
}

// webScript scripts the non-Spotify web boundaries: the lyrics providers,
// the search providers and artwork downloads.
type webScript struct {
	lyricsErr   bool // lrclib and genius both fail
	noLyrics    bool // lrclib answers with empty lyrics
	artworkErr  bool // the artwork URL answers 404
	providerErr bool // youtube and qobuz both fail
	noResults   bool // providers answer successfully but find nothing
	qobuzEmpty  bool // qobuz finds nothing, youtube carries the search
}

// installWeb installs the Spotify router plus the scripted web fixtures
// as the default transport.
func installWeb(t *testing.T, overrides map[string]func(*http.Request) (int, string), web webScript) {
	t.Helper()
	router := newSpotifyRouter(overrides)
	installTransport(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Host {
		case "accounts.spotify.com", "api.spotify.com":
			return router.RoundTrip(request)
		case "lrclib.net":
			return lrclibResponse(web)
		case "api.genius.com":
			return geniusResponse(web), nil
		case "www.youtube.com":
			return youtubeResponse(web)
		case "open.qobuz.com":
			return qobuzBundleResponse(web, request)
		case "www.qobuz.com":
			return qobuzSearchResponse(web), nil
		case "dabmusic.xyz":
			return dabResponse(web)
		case "ima.ge":
			// the artwork blob download
			if web.artworkErr {
				return rawResponse(404, "text/plain", []byte("not found")), nil
			}
			return rawResponse(200, "image/jpeg", jpegBytes(t)), nil
		default:
			// any other host is unreachable: in particular, blob
			// support probes (HEAD on a video page URL) must fail, so
			// downloads fall through to the downloader owning the URL
			return rawResponse(404, "text/plain", []byte("not found")), nil
		}
	}))
}

func lrclibResponse(web webScript) (*http.Response, error) {
	if web.lyricsErr {
		return nil, errors.New("ko")
	}
	if web.noLyrics {
		return jsonResponse(200, `{"plainLyrics":"","syncedLyrics":""}`), nil
	}
	return jsonResponse(200, `{"plainLyrics":"some lyrics","syncedLyrics":""}`), nil
}

func geniusResponse(web webScript) *http.Response {
	if web.lyricsErr {
		return jsonResponse(500, `{}`)
	}
	return jsonResponse(200, `{"response":{"hits":[]}}`)
}

func youtubeResponse(web webScript) (*http.Response, error) {
	if web.providerErr {
		return nil, errors.New("ko")
	}
	if web.noResults {
		return rawResponse(200, "text/html", []byte("<html></html>")), nil
	}
	return rawResponse(200, "text/html", []byte(youtubePage())), nil
}

func qobuzBundleResponse(web webScript, request *http.Request) (*http.Response, error) {
	if web.providerErr {
		return nil, errors.New("ko")
	}
	if strings.HasSuffix(request.URL.Path, "/track/1") {
		return rawResponse(200, "text/html", []byte(`<script src="/resources/1.0/js/main.js"></script>`)), nil
	}
	return rawResponse(200, "text/javascript", []byte(`var c={app_id:"123456789",app_secret:"00000000000000000000000000000000"};`)), nil
}

func qobuzSearchResponse(web webScript) *http.Response {
	if web.noResults || web.qobuzEmpty {
		return jsonResponse(200, `{"tracks":{"items":[]}}`)
	}
	return jsonResponse(200, `{"tracks":{"items":[{"id":138731318,"performer":{"name":"Artist"}}]}}`)
}

func dabResponse(web webScript) (*http.Response, error) {
	if web.providerErr {
		return nil, errors.New("ko")
	}
	return jsonResponse(200, `{"url":"https://cdn.qobuz.example/track.mp3"}`), nil
}

// jpegBytes encodes a real 1x1 JPEG: the artwork pipeline decodes the
// downloaded picture, so placeholder bytes would not pass.
func jpegBytes(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	assert.Nil(t, jpeg.Encode(&buffer, img, nil))
	return buffer.Bytes()
}

// youtubePage renders a minimal YouTube results page carrying one video
// matching the fixture track.
func youtubePage() string {
	return `<script>var ytInitialData = {"contents":{"twoColumnSearchResultsRenderer":` +
		`{"primaryContents":{"sectionListRenderer":{"contents":[{"itemSectionRenderer":` +
		`{"contents":[{"videoRenderer":{"videoId":"abc123","title":{"runs":[{"text":"Title"}]},` +
		`"ownerText":{"runs":[{"text":"Artist"}]},` +
		`"viewCountText":{"simpleText":"1.000.000 views"},` +
		`"lengthText":{"simpleText":"3:00 minutes"},` +
		`"publishedTimeText":{"simpleText":"1 year ago"}}}]}}]}}}}}</script>`
}

// seedSession writes a valid, non-expired OAuth2 token to the session
// file the spotify package reads at startup, and sets the credentials the
// authenticator is built from, so spotify.Authenticate recovers the
// session without any browser flow.
func seedSession(t *testing.T) {
	t.Helper()
	t.Setenv("SPOTIFY_ID", "test-id")
	t.Setenv("SPOTIFY_KEY", "test-secret")

	path := sys.CacheFile(spotify.TokenBasename)
	assert.Nil(t, os.MkdirAll(filepath.Dir(path), 0o755))
	token := `{"access_token":"access","token_type":"Bearer",` +
		`"refresh_token":"refresh","expiry":"2999-01-01T00:00:00Z"}`
	assert.Nil(t, os.WriteFile(path, []byte(token), 0o600))
	t.Cleanup(func() { _ = os.Remove(path) })
}

// noSession makes sure no recoverable session exists and credentials are
// set, so Authenticate falls through to the interactive browser flow.
func noSession(t *testing.T) {
	t.Helper()
	t.Setenv("SPOTIFY_ID", "test-id")
	t.Setenv("SPOTIFY_KEY", "test-secret")
	path := sys.CacheFile(spotify.TokenBasename)
	_ = os.Remove(path)
	t.Cleanup(func() { _ = os.Remove(path) })
}

// fakeBinDir installs fake ffmpeg, yt-dlp and opener binaries on PATH and
// returns their directory. A binary's behavior is driven by control files
// in the same directory: creating "<name>.fail" makes it fail, and the
// opener appends its arguments to "open.log".
func fakeBinDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	ffmpeg := `#!/bin/sh
if [ -f "` + dir + `/ffmpeg.fail" ]; then echo "ffmpeg blew up" >&2; exit 1; fi
case "$*" in
*measured_I*)
  if [ -f "` + dir + `/ffmpeg.normfail" ]; then echo "normalize blew up" >&2; exit 1; fi
  ;;
*print_format=json*)
  if [ -f "` + dir + `/ffmpeg.badjson" ]; then
    echo "no loudness summary here" >&2
  elif [ -f "` + dir + `/ffmpeg.far" ]; then
    printf '%s\n' '{"input_i":"-30.00","input_tp":"-1.50","input_lra":"11.00","input_thresh":"-25.00","target_offset":"16.00"}' >&2
  else
    printf '%s\n' '{"input_i":"-14.00","input_tp":"-1.50","input_lra":"11.00","input_thresh":"-25.00","target_offset":"0.00"}' >&2
  fi
  ;;
esac
last=""
for arg in "$@"; do last="$arg"; done
case "$last" in
null|"") ;;
*) printf 'audio' > "$last" ;;
esac
exit 0
`
	ytdlp := `#!/bin/sh
printf '%s\n' "$*" >> "` + dir + `/ytdlp.log"
if [ -f "` + dir + `/ytdlp.fail" ]; then echo "yt-dlp blew up" >&2; exit 1; fi
out=""
fmt=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "--output" ]; then out="$arg"; fi
  if [ "$prev" = "--audio-format" ]; then fmt="$arg"; fi
  prev="$arg"
done
path=$(printf '%s' "$out" | sed "s/%(ext)s/$fmt/")
: > "$path"
exit 0
`
	opener := `#!/bin/sh
printf '%s\n' "$*" >> "` + dir + `/open.log"
if [ -f "` + dir + `/open.fail" ]; then exit 1; fi
exit 0
`
	for name, script := range map[string]string{
		"ffmpeg": ffmpeg, "yt-dlp": ytdlp,
		"xdg-open": opener, "open": opener, "rundll32": opener,
	} {
		assert.Nil(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

// noOpenerPath restricts PATH to an empty directory, so the platform
// opener binary cannot be started at all: sys/cmd.Open uses Start
// without Wait, so a failing opener would otherwise look successful
// and the browser flow would wait for a callback forever.
func noOpenerPath(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

// feedStdin replaces os.Stdin with a pipe carrying the given input, for
// the interactive prompts of manual-mode syncs.
func feedStdin(t *testing.T, input string) {
	t.Helper()
	reader, writer, err := os.Pipe()
	assert.Nil(t, err)
	previous := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = previous
		_ = reader.Close()
	})
	_, err = writer.WriteString(input)
	assert.Nil(t, err)
	assert.Nil(t, writer.Close())
}

// occupyAuthPort binds the OAuth2 loopback port for the duration of the
// test. Authenticate's nursery waits for every job, so a flow whose
// callback never arrives would hang forever; a server that cannot bind
// fails the flow fast and deterministically instead.
func occupyAuthPort(t *testing.T) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:65535")
	assert.Nil(t, err)
	t.Cleanup(func() { _ = listener.Close() })
}

// resetState restores the package-level sync state between command runs.
func resetState(t *testing.T) {
	t.Helper()
	t.Cleanup(cleanup)
	cleanup()
}

// clearTrackCache removes the per-track cache files (download, artwork,
// lyrics) for the given ids, so cached state never leaks between tests.
func clearTrackCache(t *testing.T, ids ...string) {
	t.Helper()
	for _, id := range ids {
		track := &entity.Track{ID: id, Artwork: entity.Artwork{URL: "http://ima.ge"}}
		_ = os.Remove(track.Path().Download())
		_ = os.Remove(track.Path().Artwork())
		_ = os.Remove(track.Path().Lyrics())
	}
}
