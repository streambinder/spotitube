package spotify

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	spttb_system "system"

	api "github.com/zmb3/spotify"
)

var (
	clientChannel       = make(chan *api.Client)
	clientState         = spttb_system.RandString(20)
	clientAuthenticator = api.NewAuthenticator(
		spttb_system.SpotifyRedirectURL,
		api.ScopeUserLibraryRead,
		api.ScopePlaylistReadPrivate,
		api.ScopePlaylistReadCollaborative)
)

// Spotify : struct object containing all the informations needed to authenticate and fetch from Spotify
type Spotify struct {
	Client *api.Client
}

// AuthURL : generate new authentication URL
func AuthURL() string {
	clientAuthenticator.SetAuthInfo(spttb_system.SpotifyClientID, spttb_system.SpotifyClientSecret)
	return clientAuthenticator.AuthURL(clientState)
}

// NewClient : return a new Spotify instance
func NewClient() *Spotify {
	return &Spotify{}
}

// Auth : start local callback server to handle xdg-preferred browser authentication redirection
func (spotify *Spotify) Auth(url string) bool {
	http.HandleFunc("/favicon.ico", subHTTPFaviconHandler)
	http.HandleFunc("/callback", subHTTPCompleteAuthHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)

	commandCmd := "xdg-open"
	commandArgs := []string{url}
	_, err := exec.Command(commandCmd, commandArgs...).Output()
	if err != nil {
		return false
	}

	spotify.Client = <-clientChannel

	return true
}

// LibraryTracks : return array of Spotify FullTrack of all authenticated user library songs
func (spotify *Spotify) LibraryTracks() ([]api.FullTrack, error) {
	var (
		tracks     []api.FullTrack
		iterations int
		options    = subDefaultOptions()
	)
	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := spotify.Client.CurrentUsersTracksOpt(&options)
		if err != nil {
			return []api.FullTrack{}, fmt.Errorf(fmt.Sprintf("Something gone wrong while reading %dth chunk of tracks: %s.", iterations, err.Error()))
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

// Playlist : return Spotify FullPlaylist from input string playlistURI
func (spotify *Spotify) Playlist(playlistURI string) (*api.FullPlaylist, error) {
	playlistOwner, playlistID, playlistErr := subParsePlaylistURI(playlistURI)
	if playlistErr != nil {
		return &api.FullPlaylist{}, playlistErr
	}
	return spotify.Client.GetPlaylist(playlistOwner, playlistID)
}

// PlaylistTracks : return array of Spotify FullTrack of all input string playlistURI identified playlist
func (spotify *Spotify) PlaylistTracks(playlistURI string) ([]api.FullTrack, error) {
	var (
		tracks     []api.FullTrack
		iterations int
		options    = subDefaultOptions()
	)
	playlistOwner, playlistID, playlistErr := subParsePlaylistURI(playlistURI)
	if playlistErr != nil {
		return tracks, playlistErr
	}
	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := spotify.Client.GetPlaylistTracksOpt(playlistOwner, playlistID, &options, "")
		if err != nil {
			return []api.FullTrack{}, fmt.Errorf(fmt.Sprintf("Something gone wrong while reading %dth chunk of tracks: %s.", iterations, err.Error()))
		}
		for _, track := range chunk.Tracks {
			tracks = append(tracks, track.Track)
		}
		if len(chunk.Tracks) < 50 {
			break
		}
		iterations++
	}
	return tracks, nil
}

// Albums : return array Spotify FullAlbum, specular to the array of Spotify ID
func (spotify *Spotify) Albums(ids []api.ID) ([]api.FullAlbum, error) {
	var (
		albums     []api.FullAlbum
		iterations int
		upperbound int
		lowerbound int
	)
	for true {
		lowerbound = iterations * 20
		if upperbound = lowerbound + 20; upperbound >= len(ids) {
			upperbound = lowerbound + (len(ids) - lowerbound)
		}
		chunk, err := spotify.Client.GetAlbums(ids[lowerbound:upperbound]...)
		if err != nil {
			var chunk []api.FullAlbum
			for _, albumID := range ids[lowerbound:upperbound] {
				album, err := spotify.Client.GetAlbum(albumID)
				if err == nil {
					chunk = append(chunk, *album)
				} else {
					chunk = append(chunk, api.FullAlbum{})
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

func subDefaultOptions() api.Options {
	var (
		optLimit  = 50
		optOffset = 0
	)
	return api.Options{
		Limit:  &optLimit,
		Offset: &optOffset,
	}
}

func subParsePlaylistURI(playlistURI string) (string, api.ID, error) {
	if strings.Count(playlistURI, ":") == 4 {
		return strings.Split(playlistURI, ":")[2], api.ID(strings.Split(playlistURI, ":")[4]), nil
	}
	return "", "", fmt.Errorf(fmt.Sprintf("Malformed playlist URI: expected 5 columns, given %d.", strings.Count(playlistURI, ":")))
}

func subHTTPFaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, spttb_system.SpotifyFaviconURL, 301)
}

func subHTTPCompleteAuthHandler(w http.ResponseWriter, r *http.Request) {
	tok, err := clientAuthenticator.Token(clientState, r)
	if err != nil {
		http.Error(w, subHTTPMessage("Couldn't get token", "none"), http.StatusForbidden)
		// logger.Fatal("Couldn't get token.")
	}
	if st := r.FormValue("state"); st != clientState {
		http.NotFound(w, r)
		// logger.Fatal("\"state\" value not found.")
	}
	client := clientAuthenticator.NewClient(tok)
	fmt.Fprintf(w, subHTTPMessage("Login completed", "Come back to the shell and enjoy the magic!"))
	// logger.Log("Login process completed.")
	clientChannel <- &client
}

func subHTTPMessage(contentTitle string, contentSubtitle string) string {
	return fmt.Sprintf(spttb_system.SpotifyHTMLTemplate, contentTitle, contentSubtitle)
}
