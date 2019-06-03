package spotify

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"../system"

	"github.com/zmb3/spotify"
)

// Spotify : struct object containing all the informations needed to authenticate and fetch from Spotify
type Spotify struct {
	Client *spotify.Client
}

// AuthURL : struct object containing both the full authentication URL provided by Spotify and the shortened one using TinyURL
type AuthURL struct {
	Full  string
	Short string
}

// Playlist : alias for Spotify FullPlaylist
type Playlist = spotify.FullPlaylist

// Album : alias for Spotify FullAlbum
type Album = spotify.FullAlbum

// Track : alias for Spotify FullTrack
type Track = spotify.FullTrack

// ID : alias for Spotify ID
type ID = spotify.ID

const (
	// SpotifyClientID : Spotify app client ID
	SpotifyClientID = ":SPOTIFY_CLIENT_ID:"
	// SpotifyClientSecret : Spotify app client secret key
	SpotifyClientSecret = ":SPOTIFY_CLIENT_SECRET:"

	// SpotifyRedirectURL : Spotify app redirect URL
	SpotifyRedirectURL = "http://%s:8080/callback"
	// SpotifyFaviconURL : Spotify app redirect URL's favicon
	SpotifyFaviconURL = "https://spotify.com/favicon.ico"
	// SpotifyHTMLAutoCloseTimeout : Spotify app redirect URL's autoclose timeout
	SpotifyHTMLAutoCloseTimeout = "5" // s
	// SpotifyHTMLAutoCloseTimeoutMs : Spotify app redirect URL's autoclose timeout in ms (automatically parsed from SpotifyHTMLAutoCloseTimeout)
	SpotifyHTMLAutoCloseTimeoutMs = SpotifyHTMLAutoCloseTimeout + "000" // ms
	// SpotifyHTMLSigAuthor : Spotify app redirect URL's footer quoted author
	SpotifyHTMLSigAuthor = "streambinder"
	// SpotifyHTMLSigIcon : Spotify app redirect URL's footer quoted author icon
	SpotifyHTMLSigIcon = "data:image/png;charset=utf-8;base64,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"
	// SpotifyHTMLTemplate : Spotify app redirect URLS's template
	SpotifyHTMLTemplate = `<!DOCTYPE html>
	<html>
	<head>
		<meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\">
		<title>SpotiTube</title>
		<link rel="icon" href="` + SpotifyFaviconURL + `" type="image/x-icon" />
		<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css" />
		<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto+Condensed" />
		<style>
			body               { font: 20px 'Roboto Condensed', sans-serif; text-align: center; margin: 75px 0; padding: 50px; color: #333; border: solid 5px #1ED760; border-left: none; border-right: none; }
			body *             { text-align: center; }
			h1                 { font-size: 40px; text-transform: uppercase; }
			h1>i.fa            { margin: auto 10px; }
			article            { display: block; text-align: left; width: 650px; margin: 0 auto 50px; }
			a                  { color: #dc8100; text-decoration: none; }
			a:hover            { color: #333; text-decoration: none; }
			p.timer            { font-size: 14px; color: #A0A0A0; text-align: center; text-transform: uppercase; }
			div.signature      { border: 1px solid rgba(0, 0, 0, 0.05); border-radius: 5px; text-align: center; }
			div.signature>img  { width: 35px; vertical-align: middle; }
			div.signature>span { font-size: 15px; color: #505050; }
		</style>
		<script type="text/javascript">
			var timeleft = ` + SpotifyHTMLAutoCloseTimeout + `;
			var downloadTimer = setInterval(function() {
				timeleft--;
				document.getElementById("timer").textContent = timeleft;
				if(timeleft <= 0)
					clearInterval(downloadTimer);
			}, 1000);
			function setAutoClose() { window.setTimeout(autoClose, ` + SpotifyHTMLAutoCloseTimeoutMs + `); }
			function autoClose() { window.close(); }
		</script>
	</head>
	<body onLoad="setAutoClose()">
		<article>
			<h1><i class="fa fa-thumbs-up" aria-hidden="true"></i><br>%s</h1>
			<div>
				<h3>%s</h3>
				<br><br><br>
				<p class="timer">Window will attempt to close in <span id="timer">` + SpotifyHTMLAutoCloseTimeout + `</span> seconds.</p>
				<br>
				<div class="signature">
					<img src="` + SpotifyHTMLSigIcon + `"/>
					<span>` + SpotifyHTMLSigAuthor + `</span>
				</div>
			</div>
		</article>
	</body>
	</html>`
)

var (
	clientChannel       = make(chan *spotify.Client)
	clientState         = system.RandString(20)
	clientAuthenticator spotify.Authenticator
)

// BuildAuthURL : generate new authentication URL
func BuildAuthURL(callbackHost string) *AuthURL {
	var (
		spotifyID  = os.Getenv("SPOTIFY_ID")
		spotifyKey = os.Getenv("SPOTIFY_KEY")
	)
	if len(spotifyID) == 0 {
		spotifyID = SpotifyClientID
	}
	if len(spotifyKey) == 0 {
		spotifyKey = SpotifyClientSecret
	}
	clientAuthenticator = authenticator(fmt.Sprintf(SpotifyRedirectURL, callbackHost))
	clientAuthenticator.SetAuthInfo(spotifyID, spotifyKey)
	spotifyURL := clientAuthenticator.AuthURL(clientState)
	tinyURL := fmt.Sprintf("http://tinyurl.com/api-create.php?url=%s", spotifyURL)
	tinyResponse, tinyErr := http.Get(tinyURL)
	if tinyErr != nil {
		return &AuthURL{Full: spotifyURL, Short: ""}
	}
	defer tinyResponse.Body.Close()
	tinyContent, tinyErr := ioutil.ReadAll(tinyResponse.Body)
	if tinyErr != nil {
		return &AuthURL{Full: spotifyURL, Short: ""}

	}
	return &AuthURL{Full: spotifyURL, Short: string(tinyContent)}
}

// NewClient : return a new Spotify instance
func NewClient() *Spotify {
	return &Spotify{}
}

// Auth : start local callback server to handle xdg-preferred browser authentication redirection
func (s *Spotify) Auth(url string, authHost string, xdgOpen bool) bool {
	var authBind string
	if strings.Contains(authHost, "127.0.0.1") || strings.Contains(authHost, "localhost") {
		authBind = authHost
	} else {
		authBind = "0.0.0.0"
	}
	authServer := &http.Server{Addr: fmt.Sprintf("%s:8080", authBind)}
	http.HandleFunc("/favicon.ico", webHTTPFaviconHandler)
	http.HandleFunc("/callback", webHTTPCompleteAuthHandler)

	go func() {
		if err := authServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()

	if xdgOpen {
		var commandCmd string
		if runtime.GOOS == "windows" {
			commandCmd = "start"
		} else {
			commandCmd = "xdg-open"
		}
		commandArgs := []string{url}
		_, err := exec.Command(commandCmd, commandArgs...).Output()
		if err != nil {
			return false
		}
	}

	s.Client = <-clientChannel
	if authServer != nil {
		authServer.Shutdown(context.Background())
	}

	return true
}

// User : get authenticated username from authenticated client
func (s *Spotify) User() (string, string) {
	if user, err := s.Client.CurrentUser(); err == nil {
		return user.DisplayName, user.ID
	}
	return "unknown", "unknown"
}

// LibraryTracks : return array of Spotify FullTrack of all authenticated user library songs
func (s *Spotify) LibraryTracks() ([]Track, error) {
	var (
		tracks     []Track
		iterations int
		options    = defaultOptions()
	)
	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := s.Client.CurrentUsersTracksOpt(&options)
		if err != nil {
			return []Track{}, fmt.Errorf(fmt.Sprintf("Something gone wrong while reading %dth chunk of tracks: %s.", iterations, err.Error()))
		}
		for _, track := range chunk.Tracks {
			tracks = append(tracks, track.FullTrack)
		}
		if len(chunk.Tracks) < 50 {
			break
		}
		iterations++
	}
	return tracks, nil
}

// RemoveLibraryTracks : remove an array of tracks by their IDs from library
func (s *Spotify) RemoveLibraryTracks(ids []ID) error {
	if len(ids) == 0 {
		return nil
	}

	var iterations int
	for true {
		lowerbound := iterations * 50
		upperbound := lowerbound + 50
		if len(ids) < upperbound {
			upperbound = lowerbound + (len(ids) - lowerbound)
		}
		chunk := ids[lowerbound:upperbound]
		if err := s.Client.RemoveTracksFromLibrary(chunk...); err != nil {
			return fmt.Errorf(fmt.Sprintf("Something gone wrong while removing %dth chunk of removing tracks: %s.", iterations, err.Error()))
		}
		if len(chunk) < 50 {
			break
		}
		iterations++
	}
	return nil
}

// Playlist : return Spotify FullPlaylist from input string playlistURI
func (s *Spotify) Playlist(playlistURI string) (*Playlist, error) {
	_, playlistID, playlistErr := parsePlaylistURI(playlistURI)
	if playlistErr != nil {
		return &Playlist{}, playlistErr
	}
	return s.Client.GetPlaylist(playlistID)
}

// PlaylistTracks : return array of Spotify FullTrack of all input string playlistURI identified playlist
func (s *Spotify) PlaylistTracks(playlistURI string) ([]Track, error) {
	var (
		tracks     []Track
		iterations int
		options    = defaultOptions()
	)
	_, playlistID, playlistErr := parsePlaylistURI(playlistURI)
	if playlistErr != nil {
		return tracks, playlistErr
	}
	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := s.Client.GetPlaylistTracksOpt(playlistID, &options, "")
		if err != nil {
			return []Track{}, fmt.Errorf(fmt.Sprintf("Something gone wrong while reading %dth chunk of tracks: %s.", iterations, err.Error()))
		}
		for _, track := range chunk.Tracks {
			if !track.IsLocal {
				tracks = append(tracks, track.Track)
			}
		}
		if len(chunk.Tracks) < 50 {
			break
		}
		iterations++
	}
	return tracks, nil
}

// RemovePlaylistTracks : remove an array of tracks by their IDs from playlist
func (s *Spotify) RemovePlaylistTracks(playlistURI string, ids []ID) error {
	if len(ids) == 0 {
		return nil
	}

	_, playlistID, playlistErr := parsePlaylistURI(playlistURI)
	if playlistErr != nil {
		return playlistErr
	}
	var (
		iterations int
	)
	for true {
		lowerbound := iterations * 50
		upperbound := lowerbound + 50
		if len(ids) < upperbound {
			upperbound = lowerbound + (len(ids) - lowerbound)
		}
		chunk := ids[lowerbound:upperbound]
		if _, err := s.Client.RemoveTracksFromPlaylist(playlistID, chunk...); err != nil {
			return fmt.Errorf(fmt.Sprintf("Something gone wrong while removing %dth chunk of removing tracks: %s.", iterations, err.Error()))
		}
		if len(chunk) < 50 {
			break
		}
		iterations++
	}
	return nil
}

// Albums : return array Spotify FullAlbum, specular to the array of Spotify ID
func (s *Spotify) Albums(ids []ID) ([]Album, error) {
	var (
		albums     []spotify.FullAlbum
		iterations int
		upperbound int
		lowerbound int
	)
	for true {
		lowerbound = iterations * 20
		if upperbound = lowerbound + 20; upperbound >= len(ids) {
			upperbound = lowerbound + (len(ids) - lowerbound)
		}
		chunk, err := s.Client.GetAlbums(ids[lowerbound:upperbound]...)
		if err != nil {
			var chunk []spotify.FullAlbum
			for _, albumID := range ids[lowerbound:upperbound] {
				album, err := s.Client.GetAlbum(albumID)
				if err == nil {
					chunk = append(chunk, *album)
				} else {
					chunk = append(chunk, spotify.FullAlbum{})
				}
			}
		}
		for _, album := range chunk {
			albums = append(albums, *album)
		}
		if len(chunk) < 20 {
			break
		}
		iterations++
	}
	return albums, nil
}

func authenticator(callbackURI string) spotify.Authenticator {
	return spotify.NewAuthenticator(
		callbackURI,
		spotify.ScopeUserLibraryRead,
		spotify.ScopeUserLibraryModify,
		spotify.ScopePlaylistReadPrivate,
		spotify.ScopePlaylistReadCollaborative,
		spotify.ScopePlaylistModifyPublic,
		spotify.ScopePlaylistModifyPrivate)
}

func defaultOptions() spotify.Options {
	var (
		optLimit  = 50
		optOffset = 0
	)
	return spotify.Options{
		Limit:  &optLimit,
		Offset: &optOffset,
	}
}

func parsePlaylistURI(playlistURI string) (string, ID, error) {
	if strings.Count(playlistURI, ":") == 4 {
		return strings.Split(playlistURI, ":")[2], ID(strings.Split(playlistURI, ":")[4]), nil
	}
	return "", "", fmt.Errorf(fmt.Sprintf("Malformed playlist URI: expected 5 columns, given %d.", strings.Count(playlistURI, ":")))
}

func webHTTPFaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, SpotifyFaviconURL, 301)
}

func webHTTPCompleteAuthHandler(w http.ResponseWriter, r *http.Request) {
	tok, err := clientAuthenticator.Token(clientState, r)
	if err != nil {
		http.Error(w, webHTTPMessage("Couldn't get token", "none"), http.StatusForbidden)
		// logger.Fatal("Couldn't get token.")
	}
	if st := r.FormValue("state"); st != clientState {
		http.NotFound(w, r)
		// logger.Fatal("\"state\" value not found.")
	}
	client := clientAuthenticator.NewClient(tok)
	fmt.Fprintf(w, webHTTPMessage("Login completed", "Come back to the shell and enjoy the magic!"))
	// logger.Log("Login process completed.")
	clientChannel <- &client
}

func webHTTPMessage(contentTitle string, contentSubtitle string) string {
	return fmt.Sprintf(SpotifyHTMLTemplate, contentTitle, contentSubtitle)
}
