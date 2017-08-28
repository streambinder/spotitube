package youtube

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/kennygrant/sanitize"
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
	logger.Log("Searching for youtube results related to \"" + track.Filename + "\".")
	url, err := UrlFor(track)
	if err != nil {
		return err
	}
	logger.Log("Parsing youtube result ID from URL \"" + url + "\".")
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

func UrlFor(track Track) (string, error) {
	doc, err := goquery.NewDocument(fmt.Sprintf(YOUTUBE_QUERY_PATTERN, sanitize.Path(track.SearchPattern)))
	if err != nil {
		logger.Fatal("Cannot retrieve doc from \"" + fmt.Sprintf(YOUTUBE_QUERY_PATTERN, sanitize.Path(track.SearchPattern)) + "\": " + err.Error())
		return "", err
	}
	selection := doc.Find(YOUTUBE_VIDEO_SELECTOR)
	for lap, _ := range [2]int{} {
		for selection_item := range selection.Nodes {
			item := selection.Eq(selection_item)
			if item_href, item_href_ok := item.Attr("href"); item_href_ok {
				if item_title, item_title_ok := item.Attr("title"); item_title_ok {
					if !(strings.Contains(item_href, "&list=") || strings.Contains(item_href, "/user/")) &&
						!strings.Contains(strings.ToLower(item_title), " cover") &&
						track.Seems(item_title) {
						if strings.Contains(strings.ToLower(item_title), "official video") && lap == 0 {
							logger.Log("First page readup, temporarily ignoring \"" + item_title + "\".")
							continue
						}
						logger.Log("Video \"" + item_title + "\" matches with track \"" + track.Artist + " - " + track.Title + "\".")
						return YOUTUBE_VIDEO_PREFIX + item_href, nil
						break

					}
				}
			} else {
				logger.Log("YouTube video url (from href attr) not found. Continuing scraping...")
			}
		}
	}

	logger.Log("YouTube video url (from href attr) not found. Dropping song download.")
	return "", errors.New("YouTube video url not found")
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
	command_args := []string{"--output", track.Track.FilenameTemp + ".%(ext)s", "--format", "bestaudio", "--extract-audio", "--audio-format", track.Track.FilenameExt[1:], "--audio-quality", "0", track.URL}
	_, err := exec.Command(command_cmd, command_args...).Output()
	if err != nil {
		logger.Fatal("Something went wrong while executing \"" + command_cmd + " " + strings.Join(command_args, " ") + "\": " + err.Error())
		return err
	}
	logger.Log("Song downloaded to: \"" + download_path + "/" + track.Track.FilenameTemp + track.Track.FilenameExt + "\".")

	return nil
}
