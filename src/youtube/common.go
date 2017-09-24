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
	Title string
	User  string
}

func FetchAndDownload(track Track, path string) error {
	download_path = path
	youtube_track, err := FindTrack(track)
	if err != nil {
		return err
	}
	err = youtube_track.Download()
	if err != nil {
		return err
	}
	return nil
}

func FindTrack(track Track) (YouTubeTrack, error) {
	logger.Log("Searching youtube results to \"" + fmt.Sprintf(YOUTUBE_QUERY_PATTERN, strings.Replace(track.SearchPattern, " ", "+", -1)) + "\".")
	doc, err := goquery.NewDocument(fmt.Sprintf(YOUTUBE_QUERY_PATTERN, strings.Replace(track.SearchPattern, " ", "+", -1)))
	if err != nil {
		logger.Warn("Cannot retrieve doc from \"" + fmt.Sprintf(YOUTUBE_QUERY_PATTERN, strings.Replace(track.SearchPattern, " ", "+", -1)) + "\": " + err.Error())
		return YouTubeTrack{}, err
	}
	selection := doc.Find(YOUTUBE_VIDEO_SELECTOR)
	selection_desc := doc.Find(YOUTUBE_DESC_SELECTOR)
	for lap := range [2]int{} {
		for selection_item := range selection.Nodes {
			item := selection.Eq(selection_item)
			item_href, item_href_ok := item.Attr("href")
			item_title, item_title_ok := item.Attr("title")
			item_user, item_user_ok := "", false
			if selection_item < len(selection_desc.Nodes) {
				item_desc := selection_desc.Eq(selection_item)
				item_user = item_desc.Text()
				item_user_ok = true
			}
			if !(item_href_ok && item_title_ok && item_user_ok) {
				logger.Log("Non-standard YouTube video entry structure. Continuing scraping...")
				continue
			} else if !strings.Contains(strings.ToLower(item_href), "youtu.be") &&
				!strings.Contains(strings.ToLower(item_href), "watch?v=") {
				logger.Log("Advertising URL found. Continuing scraping...")
				continue
			}

			youtube_track := YouTubeTrack{
				Track: track,
				ID:    IdFromUrl(YOUTUBE_VIDEO_PREFIX + item_href),
				URL:   YOUTUBE_VIDEO_PREFIX + item_href,
				Title: item_title,
				User:  item_user,
			}

			logger.Debug("ID: " + youtube_track.ID +
				" | URL: " + youtube_track.URL +
				" | Title: " + youtube_track.Title +
				" | User: " + youtube_track.User)

			if (lap == 0 && youtube_track.Match(track, true)) ||
				(lap == 1 && youtube_track.Match(track, false)) {
				return youtube_track, nil
			}
		}
	}

	logger.Warn("YouTube video URL not found. Dropping song download.")
	return YouTubeTrack{}, errors.New("YouTube video URL not found")
}

func IdFromUrl(url string) string {
	var id_part string
	if strings.Contains(strings.ToLower(url), "youtu.be/") {
		id_part = strings.Split(url, "youtu.be/")[1]
	} else {
		id_part = strings.Split(url, "watch?v=")[1]
	}
	if strings.Contains(id_part, "?") {
		id_part = strings.Split(id_part, "?")[0]
	}
	if strings.Contains(id_part, "&list") {
		id_part = strings.Split(id_part, "&list")[0]
	}
	return id_part
}

func (youtube_track YouTubeTrack) Match(track Track, strict bool) bool {
	item_title := strings.ToLower(youtube_track.Title)

	if strings.Contains(youtube_track.URL, "&list=") || strings.Contains(youtube_track.URL, "/user/") {
		logger.Debug("Track is actually pointing to playlist or user.")
		return false
	} else if track.Seems(youtube_track.Title) {
		logger.Debug("Song seems that one: checking for VEVO.")
		if strict &&
			(strings.Contains(item_title, "official video") ||
				(strings.Contains(youtube_track.User, "VEVO") &&
					!(strings.Contains(item_title, "audio") || strings.Contains(item_title, "lyric")))) {
			logger.Debug("First page readup, temporarily ignoring \"" + youtube_track.Title + "\" by \"" + youtube_track.User + "\".")
			return false
		}
		logger.Log("Video \"" + youtube_track.Title + "\" matches with track \"" + track.Artist + " - " + track.Title + "\".")
		return true
	}

	return false
}

func (track YouTubeTrack) Download() error {
	os.Chdir(download_path)
	for _, filename := range []string{track.Track.FilenameTemp, track.Track.FilenameTemp + track.Track.FilenameExt, track.Track.Filename, track.Track.Filename + track.Track.FilenameExt} {
		os.Remove(filename)
	}
	logger.Log("Going to download \"" + track.URL + "\" to \"" + track.Track.FilenameTemp + track.Track.FilenameExt + "\".")
	command_cmd := "youtube-dl"
	command_args := []string{"--output", track.Track.FilenameTemp + ".%(ext)s", "--format", "bestaudio", "--extract-audio", "--audio-format", track.Track.FilenameExt[1:], "--audio-quality", "0", track.URL}
	_, err := exec.Command(command_cmd, command_args...).Output()
	if err != nil {
		logger.Warn("Something went wrong while executing \"" + command_cmd + " " + strings.Join(command_args, " ") + "\": " + err.Error())
		return err
	}
	logger.Log("Song downloaded to: \"" + download_path + "/" + track.Track.FilenameTemp + track.Track.FilenameExt + "\".")

	return nil
}
