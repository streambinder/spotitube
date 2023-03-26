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

func (youTubeDl) Supports(url string) bool {
	return strings.Contains(url, "://youtu.be") || strings.Contains(url, "://youtube.com")
}

func (youTubeDl) Download(url, path string) error {
	return cmd.YouTubeDl(url, path)
}
