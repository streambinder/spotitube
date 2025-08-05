package downloader

import (
	"strings"

	"github.com/streambinder/spotitube/processor"
	"github.com/streambinder/spotitube/sys/cmd"
)

type youTubeDl struct {
	Downloader
}

func init() {
	downloaders = append(downloaders, youTubeDl{})
}

func (youTubeDl) supports(url string) bool {
	return strings.Contains(url, "://youtu.be") || strings.Contains(url, "://www.youtube.com")
}

func (youTubeDl) download(url, path string, _ processor.Processor, channels ...chan []byte) error {
	// in this case, data won't be passed through channels
	// as too heavy
	for _, ch := range channels {
		ch <- nil
	}

	return cmd.YouTubeDl(url, path)
}
