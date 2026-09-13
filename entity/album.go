package entity

type Album struct {
	ID      string
	Name    string
	Artists []string
	Tracks  []*Track
}
