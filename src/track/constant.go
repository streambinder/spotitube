package track

const (
	// GeniusAccessToken : Genius app access token
	GeniusAccessToken = ":GENIUS_TOKEN:"
	// LyricsGeniusAPIURL : lyrics Genius API URL
	LyricsGeniusAPIURL = "https://api.genius.com/search?q=%s+%s"
	// LyricsOVHAPIURL : lyrics OVH API URL
	LyricsOVHAPIURL = "https://api.lyrics.ovh/v1/%s/%s"

	// SongTypeAlbum : identifier for Song in its album variant
	SongTypeAlbum = iota
	// SongTypeLive : identifier for Song in its live variant
	SongTypeLive
	// SongTypeCover : identifier for Song in its cover variant
	SongTypeCover
	// SongTypeRemix : identifier for Song in its remix variant
	SongTypeRemix
	// SongTypeAcoustic : identifier for Song in its acoustic variant
	SongTypeAcoustic
	// SongTypeKaraoke : identifier for Song in its karaoke variant
	SongTypeKaraoke
	// SongTypeParody : identifier for Song in its parody variant
	SongTypeParody
	// SongTypeReverse : identifier for Song in its reverse variant
	SongTypeReverse
	_
	// ID3FrameTitle : ID3 title frame tag identifier
	ID3FrameTitle = iota
	// ID3FrameSong : ID3 song frame tag identifier
	ID3FrameSong
	// ID3FrameArtist : ID3 artist frame tag identifier
	ID3FrameArtist
	// ID3FrameAlbum : ID3 album frame tag identifier
	ID3FrameAlbum
	// ID3FrameGenre : ID3 genre frame tag identifier
	ID3FrameGenre
	// ID3FrameYear : ID3 year frame tag identifier
	ID3FrameYear
	// ID3FrameFeaturings : ID3 featurings frame tag identifier
	ID3FrameFeaturings
	// ID3FrameTrackNumber : ID3 track number frame tag identifier
	ID3FrameTrackNumber
	// ID3FrameTrackTotals : ID3 total tracks number frame tag identifier
	ID3FrameTrackTotals
	// ID3FrameArtwork : ID3 artwork frame tag identifier
	ID3FrameArtwork
	// ID3FrameArtworkURL : ID3 artwork URL frame tag identifier
	ID3FrameArtworkURL
	// ID3FrameLyrics : ID3 lyrics frame tag identifier
	ID3FrameLyrics
	// ID3FrameYouTubeURL : ID3 youtube URL frame tag identifier
	ID3FrameYouTubeURL
	// ID3FrameDuration : ID3 duration frame tag identifier
	ID3FrameDuration
)
