package track

import (
	"strconv"

	"../system"
)

// Playlist defines a playlist wrapper
type Playlist struct {
	Tracks []*Track
	Name   string
	Owner  string
}

// M3U returns the M3U-compliant representation of the playlist
func (p *Playlist) M3U() string {
	content := "#EXTM3U\n"
	for i := len(p.Tracks) - 1; i >= 0; i-- {
		t := p.Tracks[i]
		if system.FileExists(t.Filename()) {
			content += "#EXTINF:" + strconv.Itoa(t.Duration) + "," + t.Filename() + "\n" +
				"./" + t.Filename() + "\n"
		}
	}

	return content
}

// PLS returns the PLS-compliant representation of the playlist
func (p *Playlist) PLS() string {
	content := "[" + p.Name + "]\n"
	for i := len(p.Tracks) - 1; i >= 0; i-- {
		t := p.Tracks[i]
		iInverse := len(p.Tracks) - i
		if system.FileExists(t.Filename()) {
			content += "File" + strconv.Itoa(iInverse) + "=./" + t.Filename() + "\n" +
				"Title" + strconv.Itoa(iInverse) + "=" + t.Filename() + "\n" +
				"Length" + strconv.Itoa(iInverse) + "=" + strconv.Itoa(t.Duration) + "\n\n"
		}
	}
	content += "NumberOfEntries=" + strconv.Itoa(len(p.Tracks)) + "\n"

	return content
}
