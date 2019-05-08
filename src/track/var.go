package track

var (
	// SongTypes : array containing every song variant identifier
	SongTypes = []int{SongTypeLive, SongTypeCover, SongTypeRemix,
		SongTypeAcoustic, SongTypeKaraoke, SongTypeParody}
	// JunkSuffixes : array containing every file suffix considered junk
	JunkSuffixes = []string{".ytdl", ".webm", ".opus", ".part", ".jpg", ".tmp", "-id3v2"}
)
