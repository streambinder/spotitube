package spotitube

import (
	"fmt"
	"os/user"
	"time"
)

const (
	// Version is current version
	Version = 25
	// TracksCacheDuration indicates lifetime of tracks cache
	TracksCacheDuration = 30 * time.Minute
	// ConcurrencyLimit indicates max concurrent jobs number
	ConcurrencyLimit = 100
	// SongExtension is the default songs extension
	SongExtension = "mp3"
)

var (
	// UserBinary indicates the path of the cached and updated application binary
	UserBinary = fmt.Sprintf("%s/spotitube", UserPath())
	// UserIndex indicates the path of the library index
	UserIndex = fmt.Sprintf("%s/index.gob", UserPath())
	// UserGob indicates the pattern of the path to a generic cache GOB file
	UserGob = fmt.Sprintf("%s/%s_%s.gob", UserPath(), "%s", "%s")
)

// UserPath : get local configuration and cache path
func UserPath() string {
	currentUser, _ := user.Current()
	return fmt.Sprintf("%s/.cache/spotitube", currentUser.HomeDir)
}
