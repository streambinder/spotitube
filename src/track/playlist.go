package track

import (
	"fmt"
	"os"
	"strconv"

	"github.com/streambinder/spotitube/src/system"
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
	for i := len(p.Tracks) - 1; i >= 0; i-- {
		t := p.Tracks[i]
		if system.FileExists(t.Filename()) {
			content += fmt.Sprintf("#EXTINF:%s,%s\n%s%c%s\n",
				strconv.Itoa(t.Duration),
				t.Filename(),
				prefix,
				os.PathSeparator,
				t.Filename())
		}
	}

	return content
}

// PLS returns the PLS-compliant representation of the playlist
func (p *Playlist) PLS(prefix string) string {
	content := fmt.Sprintf("[%s]\n", p.Name)
	for i := len(p.Tracks) - 1; i >= 0; i-- {
		t := p.Tracks[i]
		iInverse := len(p.Tracks) - i
		if system.FileExists(t.Filename()) {
			content += fmt.Sprintf("File%s=%s%c%s\nTitle%s=%s\nLength%s=%s\n\n",
				strconv.Itoa(iInverse),
				prefix,
				os.PathSeparator,
				t.Filename(),
				strconv.Itoa(iInverse),
				t.Filename(),
				strconv.Itoa(iInverse),
				strconv.Itoa(t.Duration))
		}
	}
	content += fmt.Sprintf("NumberOfEntries=%s\n", strconv.Itoa(len(p.Tracks)))

	return content
}
