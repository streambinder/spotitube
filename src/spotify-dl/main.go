package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"spotify"
	"strconv"
	"strings"
	"sync"
	"youtube"

	id3 "github.com/mikkyang/id3-go"
	api "github.com/zmb3/spotify"
)

type Track struct {
	Title  string
	Artist string
	Album  string
}

type Tracks []Track

var (
	tracks_offline   Tracks
	tracks_online    []api.SavedTrack
	tracks_delta     []api.SavedTrack
	wait_group       sync.WaitGroup
	arg_music_folder *string
)

func Dir(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	file_stat, err := file.Stat()
	if err != nil {
		return false
	}
	return file_stat.IsDir()
}

func (tracks Tracks) Has(name string, artist string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	artist = strings.TrimSpace(strings.ToLower(artist))
	for _, track := range tracks {
		track_title := strings.TrimSpace(strings.ToLower(track.Title))
		track_artist := strings.TrimSpace(strings.ToLower(track.Artist))
		if track_title == name && track_artist == artist {
			return true
		}
	}
	return false
}

func Normalize(track api.SavedTrack) api.SavedTrack {
	// title_parts := strings.Split(track.FullTrack.SimpleTrack.Name, " - ")
	// if len(title_parts) > 1 && (strings.Contains(strings.ToLower(title_parts[len(title_parts)-1]), "live") || strings.Contains(strings.ToLower(title_parts[len(title_parts)-1]), "radio edit")) {
	// 	track.FullTrack.SimpleTrack.Name = strings.Join(title_parts[:len(title_parts)-1], " - ")
	// }
	// return track
	track.FullTrack.SimpleTrack.Name = strings.Split(track.FullTrack.SimpleTrack.Name, " - ")[0]
	return track
}

func main() {
	arg_music_folder = flag.String("music", "~/Music", "Folder to sync with music.")
	flag.Parse()
	if !(Dir(*arg_music_folder)) {
		fmt.Println("Chosen music folder does not exist:", *arg_music_folder)
		os.Exit(1)
	}

	wait_group.Add(1)
	go LocalLibrary(&wait_group)
	tracks_online = spotify.AuthAndTracks()
	wait_group.Wait()

	for _, track := range tracks_online {
		// Title:  track.FullTrack.SimpleTrack.Name,
		// Artist: (track.FullTrack.SimpleTrack.Artists[0]).Name,
		// Album:  track.FullTrack.Album.Name,
		if !tracks_offline.Has(track.FullTrack.SimpleTrack.Name, (track.FullTrack.SimpleTrack.Artists[0]).Name) {
			track_delta := Normalize(track)
			tracks_delta = append(tracks_delta, track_delta)
		}
	}

	if len(tracks_delta) > 0 {
		fmt.Println("Found " + strconv.Itoa(len(tracks_delta)) + " missing songs. Proceeding to download.")
		for _, track := range tracks_delta {
			track_file := youtube.FetchAndDownload(track.FullTrack.SimpleTrack.Name, (track.FullTrack.SimpleTrack.Artists[0]).Name, *arg_music_folder)
			if track_file == "none" {
				continue
			}
			wait_group.Add(1)
			go MetadataAndMove(track_file, track, &wait_group)
		}
		wait_group.Wait()

		fmt.Println("Done")
	} else {
		fmt.Println("No song to download.")
	}
}

func LocalLibrary(wg *sync.WaitGroup) {
	tracks_files, _ := ioutil.ReadDir(*arg_music_folder)
	for _, track_file_info := range tracks_files {
		var track_file = track_file_info.Name()
		var track_file_ext = filepath.Ext(track_file)
		var track_file_name = track_file[0 : len(track_file)-len(track_file_ext)]
		track_mp3_file, err := id3.Open(*arg_music_folder + "/" + track_file)
		var track_title string
		var track_artist string
		var track_album string
		if err == nil {
			track_title = strings.TrimSpace(track_mp3_file.Title())
			track_artist = strings.TrimSpace(track_mp3_file.Artist())
			track_album = strings.TrimSpace(track_mp3_file.Album())
			defer track_mp3_file.Close()
		} else {
			track_title = strings.Split(track_file_name, " - ")[1]
			track_artist = strings.Split(track_file_name, " - ")[0]
			track_album = "none"
		}
		track := Track{
			Title:  track_title,
			Artist: track_artist,
			Album:  track_album,
		}
		tracks_offline = append(tracks_offline, track)
	}

	wg.Done()
}

func MetadataAndMove(track_file string, track api.SavedTrack, wg *sync.WaitGroup) {
	track_mp3, err := id3.Open(track_file)
	if err != nil {
		fmt.Println("Something bad happened while opening " + track_file + ".")
	} else {
		fmt.Println("Fixing metadata for:", track.FullTrack.SimpleTrack.Name+" - "+(track.FullTrack.SimpleTrack.Artists[0]).Name)
		track_mp3.SetTitle(track.FullTrack.SimpleTrack.Name)
		track_mp3.SetArtist((track.FullTrack.SimpleTrack.Artists[0]).Name)
		track_mp3.SetAlbum(track.FullTrack.Album.Name)
		defer track_mp3.Close()
	}

	dest_file := *arg_music_folder + "/" + (strings.Split(track_file, "/")[len(strings.Split(track_file, "/"))-1])[1:]
	os.Remove(dest_file)
	err = os.Rename(track_file, dest_file)
	if err != nil {
		fmt.Println("Something went wrong while moving song from \""+track_file+"\" to \""+dest_file+"\":", err.Error())
		os.Exit(1)
	} else {
		fmt.Println("Fixed metadata and moved song to \"" + dest_file + "\"")
	}

	wg.Done()
}
