package spotify

import (
	"fmt"
	"net/http"
	"strings"

	api "github.com/zmb3/spotify"
)

func defaultOptions() api.Options {
	var (
		optLimit  = 50
		optOffset = 0
	)
	return api.Options{
		Limit:  &optLimit,
		Offset: &optOffset,
	}
}

func parsePlaylistURI(playlistURI string) (string, api.ID, error) {
	if strings.Count(playlistURI, ":") == 4 {
		return strings.Split(playlistURI, ":")[2], api.ID(strings.Split(playlistURI, ":")[4]), nil
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
