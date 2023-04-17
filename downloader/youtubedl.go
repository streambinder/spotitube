package downloader

import (
	"strings"

	"github.com/streambinder/spotitube/util/cmd"
)

type youTubeDl struct {
	Downloader
}

func init() {
	downloaders = append(downloaders, youTubeDl{})
}

func (youTubeDl) supports(url string) bool {
	return strings.Contains(url, "://youtu.be") || strings.Contains(url, "://youtube.com")
}

func (youTubeDl) download(url, path string, channels ...chan []byte) error {
	// in this case, data won't be passed through channels
	// as too heavy
	return cmd.YouTubeDl(url, path)
}
