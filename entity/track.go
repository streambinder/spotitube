package entity

type Track struct {
	ID          string
	Title       string
	Artists     []string
	Album       string
	ArtworkURL  string // URL whose content to feed the Artwork field with
	Artwork     []byte
	Duration    int // in seconds
	Lyrics      []byte
	Number      int // track number within the album
	Year        string
	UpstreamURL string // URL to the upstream blob the song's been downloaded from
}
