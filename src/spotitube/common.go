package spotitube

import (
	"fmt"
	"os/user"
)

const (
	// Version : current version
	Version = 25
	// VersionRepository : repositoy container
	VersionRepository = "https://github.com/streambinder/spotitube"
	// VersionOrigin : API repository latest version URL
	VersionOrigin = "https://api.github.com/repos/streambinder/spotitube/releases/latest"
	// VersionURL : latest version for download
	VersionURL = VersionRepository + "/releases/latest"

	// ConcurrencyLimit : max concurrent jobs
	ConcurrencyLimit = 100

	// SongExtension : default downloaded songs extension
	SongExtension = "mp3"
	// TCPCheckOrigin : default internet connection check origin
	TCPCheckOrigin = "github.com:443"
	// HTTPTimeout : default timeout for HTTP calls
	HTTPTimeout = 3 // second(s)
)

// LocalConfigPath : get local configuration and cache path
func LocalConfigPath() string {
	currentUser, _ := user.Current()
	return fmt.Sprintf("%s/.cache/spotitube", currentUser.HomeDir)
}
