package youtube

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PuerkitoBio/goquery"
	. "utils"
)

var (
	logger = NewLogger()
)

type YouTubeTrack struct {
	Title    string
	Artist   string
	ID       string
	URL      string
	Filename string
}

func FetchAndDownload(track Track, path string) string {
	logger.Log("Searching for youtube results related to " + track.Title + " by " + track.Artist + ".")
	url := UrlFor(track.Title, track.Artist)
	if url == "none" {
		return "none"
	}
	logger.Log("Parsing youtube result ID from URL.")
	id := IdFromUrl(url)
	filename := track.Artist + " - " + track.Title
	youtube_track := YouTubeTrack{
		Title:    track.Title,
		Artist:   track.Artist,
		ID:       id,
		URL:      url,
		Filename: filename,
	}
	logger.Log("Firing download procedure for " + track.Title + " by " + track.Artist + ".")
	track_filename := youtube_track.Download(path)
	return track_filename
}

func UrlFor(title string, artist string) string {
	doc, err := goquery.NewDocument(fmt.Sprintf(YOUTUBE_QUERY_PATTERN, strings.Replace(artist, " ", "+", -1)+"+"+strings.Replace(title, " ", "+", -1)))
	if err != nil {
		logger.Fatal("Cannot retrieve doc from \"" + fmt.Sprintf(YOUTUBE_QUERY_PATTERN, artist+" "+title) + "\": " + err.Error())
	}
	selection := doc.Find(YOUTUBE_VIDEO_SELECTOR)
	for selection_item := range selection.Nodes {
		item := selection.Eq(selection_item)
		href, ok := item.Attr("href")
		if ok {
			return YOUTUBE_VIDEO_PREFIX + href
			break
		} else {
			logger.Log("Youtube video url (from href attr) not found. Continuing scraping...")
		}
	}
	logger.Log("Youtube video url (from href attr) not found. Dropping song download.")
	return "none"
}

func IdFromUrl(url string) string {
	var id_part string
	if strings.Contains(strings.ToLower(url), "youtu.be/") {
		id_part = strings.Split(url, "youtu.be/")[1]
	} else {
		id_part = strings.Split(url, "watch?v=")[1]
	}
	if strings.Contains(id_part, "?") {
		return strings.Split(id_part, "?")[0]
	} else {
		return id_part
	}
}

func (track YouTubeTrack) Download(path string) string {
	os.Chdir(path)
	os.Remove("." + track.Filename)
	os.Remove("." + track.Filename + ".mp3")
	logger.Log("Proceeding to download from \"" + track.URL + "\" to \"" + track.Filename + "\".")
	command_cmd := "youtube-dl"
	command_args := []string{"-o", "." + track.Filename, "-f", "bestaudio", track.URL, "--exec", "ffmpeg -i {}  -codec:a libmp3lame -qscale:a 0 {}.mp3"}
	_, err := exec.Command(command_cmd, command_args...).Output()
	if err != nil {
		logger.Fatal("Something went wrong while executing \"" + command_cmd + strings.Join(command_args, " ") + "\": " + err.Error())
	}
	logger.Log("Song downloaded to: \"." + track.Filename + ".mp3\"")
	err = os.Remove("." + track.Filename)
	if err != nil {
		logger.Log("Something went wrong while trying to remove tmp ." + track.Filename + " .")
	}
	return path + "/" + "." + track.Filename + ".mp3"
}
