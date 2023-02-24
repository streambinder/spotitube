package entity

type Playlist struct {
	ID     string
	Name   string
	Owner  string
	Tracks []*Track
}
