package youtube

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type YouTubeTrack struct {
	Title    string
	Artist   string
	ID       string
	URL      string
	Filename string
}

func FetchAndDownload(title string, artist string, path string) string {
	url := UrlFor(title, artist)
	if url == "none" {
		return "none"
	}

	id := IdFromUrl(url)
	filename := artist + " - " + title
	track := YouTubeTrack{
		Title:    title,
		Artist:   artist,
		ID:       id,
		URL:      url,
		Filename: filename,
	}
	track_filename := track.Download(path)
	return track_filename
}

func UrlFor(title string, artist string) string {
	doc, err := goquery.NewDocument(fmt.Sprintf(YOUTUBE_QUERY_PATTERN, strings.Replace(artist, " ", "+", -1)+"+"+strings.Replace(title, " ", "+", -1)))
	if err != nil {
		fmt.Println("Cannot retrieve doc from \"" + fmt.Sprintf(YOUTUBE_QUERY_PATTERN, artist+" "+title) + "\": " + err.Error())
		os.Exit(1)
	}

	selection := doc.Find(YOUTUBE_VIDEO_SELECTOR)
	for selection_item := range selection.Nodes {
		item := selection.Eq(selection_item)
		href, ok := item.Attr("href")
		if ok {
			return YOUTUBE_VIDEO_PREFIX + href
			break
		} else {
			fmt.Printf("Youtube video url (from href attr) not found. Continuing scraping...")
		}
	}

	fmt.Printf("Youtube video url (from href attr) not found. Dropping song download.")
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
	fmt.Println("Proceeding to download from \"" + track.URL + "\" to \"" + track.Filename + "\".")
	command_cmd := "youtube-dl"
	command_args := []string{"-o", "." + track.Filename, "-f", "bestaudio", track.URL, "--exec", "ffmpeg -i {}  -codec:a libmp3lame -qscale:a 0 {}.mp3"}
	_, err := exec.Command(command_cmd, command_args...).Output()
	if err != nil {
		fmt.Println("Something went wrong while executing \""+command_cmd+strings.Join(command_args, " ")+"\":", err.Error())
		os.Exit(1)
	}
	// fmt.Print(string(command_out))

	fmt.Println("Downloaded to:", "."+track.Filename+".mp3")
	os.Remove("." + track.Filename)
	return path + "/" + "." + track.Filename + ".mp3"
}
