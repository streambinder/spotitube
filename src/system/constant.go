package system

const (
	// Version : current version
	Version = 20
	// VersionRepository : repositoy container
	VersionRepository = "https://github.com/streambinder/spotitube"
	// VersionOrigin : API repository latest version URL
	VersionOrigin = "https://api.github.com/repos/streambinder/spotitube/releases/latest"
	// VersionURL : latest version for download
	VersionURL = VersionRepository + "/releases/latest"

	// ConcurrencyLimit : max concurrent jobs
	ConcurrencyLimit = 100

	// SongExtension : default downloaded songs extension
	SongExtension = ".mp3"
	// TCPCheckOrigin : default internet connection check origin
	TCPCheckOrigin = "github.com:443"
	// HTTPTimeout : default timeout for HTTP calls
	HTTPTimeout = 3 // second(s)

	// SystemLetterBytes : random string generator characters
	SystemLetterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// SystemLetterIdxBits : random string generator bits
	SystemLetterIdxBits = 6
	// SystemLetterIdxMask : random string generator mask
	SystemLetterIdxMask = 1<<SystemLetterIdxBits - 1
	// SystemLetterIdxMax : random string generator max
	SystemLetterIdxMax = 63 / SystemLetterIdxBits
)
