package youtube

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PuerkitoBio/goquery"
	. "utils"
)

var (
	download_path string
	logger        = NewLogger()
)

type YouTubeTrack struct {
	Track Track
	ID    string
	URL   string
}

func FetchAndDownload(track Track, path string) error {
	download_path = path
	logger.Log("Searching for youtube results related to " + track.Filename + ".")
	url, err := UrlFor(track.Title, track.Artist)
	if err != nil {
		return err
	}
	logger.Log("Parsing youtube result ID from URL.")
	id := IdFromUrl(url)
	youtube_track := YouTubeTrack{
		Track: track,
		ID:    id,
		URL:   url,
	}
	logger.Log("Firing download procedure for " + track.Filename + ".")
	err = youtube_track.Download()
	if err != nil {
		return err
	}
	return nil
}

func UrlFor(title string, artist string) (string, error) {
	doc, err := goquery.NewDocument(fmt.Sprintf(YOUTUBE_QUERY_PATTERN, strings.Replace(artist, " ", "+", -1)+"+"+strings.Replace(title, " ", "+", -1)))
	if err != nil {
		logger.Fatal("Cannot retrieve doc from \"" + fmt.Sprintf(YOUTUBE_QUERY_PATTERN, artist+" "+title) + "\": " + err.Error())
		return "", err
	}
	selection := doc.Find(YOUTUBE_VIDEO_SELECTOR)
	for selection_item := range selection.Nodes {
		item := selection.Eq(selection_item)
		href, ok := item.Attr("href")
		if ok {
			if strings.Contains(href, "&list=") {
				continue
			} else {
				return YOUTUBE_VIDEO_PREFIX + href, nil
				break
			}
		} else {
			logger.Log("Youtube video url (from href attr) not found. Continuing scraping...")
		}
	}

	logger.Log("Youtube video url (from href attr) not found. Dropping song download.")
	return "", errors.New("Youtube video url not found.")
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

func (track YouTubeTrack) Download() error {
	os.Chdir(download_path)
	for _, filename := range []string{track.Track.FilenameTemp, track.Track.FilenameTemp + track.Track.FilenameExt, track.Track.Filename, track.Track.Filename + track.Track.FilenameExt} {
		os.Remove(filename)
	}
	logger.Log("Proceeding to download from \"" + track.URL + "\" to \"" + track.Track.FilenameTemp + track.Track.FilenameExt + "\".")
	command_cmd := "youtube-dl"
	command_args := []string{"-o", track.Track.FilenameTemp, "-f", "bestaudio", track.URL, "--exec", "ffmpeg -i {}  -codec:a libmp3lame -qscale:a 0 {}" + track.Track.FilenameExt}
	_, err := exec.Command(command_cmd, command_args...).Output()
	if err != nil {
		logger.Fatal("Something went wrong while executing \"" + command_cmd + " " + strings.Join(command_args, " ") + "\": " + err.Error())
		return err
	}
	logger.Log("Song downloaded to: \"" + download_path + "/" + track.Track.FilenameTemp + track.Track.FilenameExt + "\".")
	err = os.Remove(download_path + "/" + track.Track.FilenameTemp)
	if err != nil {
		logger.Log("Something went wrong while trying to remove temporary file \"" + download_path + "/" + track.Track.FilenameTemp + track.Track.FilenameExt + "\".")
		return err
	}

	return nil
}
