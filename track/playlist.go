package track

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/streambinder/spotitube/system"
)

// Playlist defines a playlist wrapper
type Playlist struct {
	Tracks []*Track
	Name   string
	Owner  string
}

// M3U returns the M3U-compliant representation of the playlist
func (p *Playlist) M3U(prefix string) string {
	content := "#EXTM3U\n"
	for _, t := range p.Tracks {
		fname := prefix + string(filepath.Separator) + t.Filename()
		if system.FileExists(fname) {
			content += fmt.Sprintf("#EXTINF:%s,%s\n%s\n",
				strconv.Itoa(t.Duration),
				t.Basename(),
				fname)
		}
	}

	return content
}

// PLS returns the PLS-compliant representation of the playlist
func (p *Playlist) PLS(prefix string) string {
	content := fmt.Sprintf("[%s]\n", p.Name)
	for i, t := range p.Tracks {
		fname := prefix + string(filepath.Separator) + t.Filename()
		if system.FileExists(fname) {
			content += fmt.Sprintf("File%s=%s\nTitle%s=%s\nLength%s=%s\n\n",
				strconv.Itoa(i+1),
				fname,
				strconv.Itoa(i+1),
				t.Basename(),
				strconv.Itoa(i+1),
				strconv.Itoa(t.Duration))
		}
	}
	content += fmt.Sprintf("NumberOfEntries=%s\n", strconv.Itoa(len(p.Tracks)))

	return content
}
