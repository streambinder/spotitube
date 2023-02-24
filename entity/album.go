package entity

type Album struct {
	ID     string
	Name   string
	Artist string
	Tracks []*Track
}
