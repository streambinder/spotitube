package spotify

import (
	"path"
	"strings"

	"github.com/zmb3/spotify/v2"
)

// resources on Spotify can be targeted in the following forms forms:
// - ID: 1234567890123456789012
// - URI: spotify:track:1234567890123456789012
// - URL: https://open.spotify.com/track/1234567890123456789012?si=abcdefghijklmnop
func id(target string) spotify.ID {
	// get last piece when split by colon (URI)
	uriParts := strings.Split(target, ":")
	target = uriParts[len(uriParts)-1]

	// get last portion as a path (URL)
	target = path.Base(target)
	// remove query parameters (URL)
	target = strings.Split(target, "?")[0]

	return spotify.ID(target)
}
