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
	. "utils"
)

var (
	tracks_offline   Tracks
	tracks_delta     Tracks
	wait_group       sync.WaitGroup
	arg_music_folder *string
)

func main() {
	arg_music_folder = flag.String("music", "~/Music", "Folder to sync with music.")
	flag.Parse()
	if !(IsDir(*arg_music_folder)) {
		fmt.Println("Chosen music folder does not exist:", *arg_music_folder)
		os.Exit(1)
	}

	wait_group.Add(1)
	go LocalLibrary(&wait_group)
	tracks_online := spotify.AuthAndTracks()
	wait_group.Wait()

	for _, track_online := range tracks_online {
		track := Track{
			Title:  track_online.FullTrack.SimpleTrack.Name,
			Artist: (track_online.FullTrack.SimpleTrack.Artists[0]).Name,
			Album:  track_online.FullTrack.Album.Name,
		}.Normalize()
		if !tracks_offline.Has(track) {
			tracks_delta = append(tracks_delta, track)
		}
	}

	if len(tracks_delta) > 0 {
		fmt.Println("Found " + strconv.Itoa(len(tracks_delta)) + " missing songs. Proceeding to download.")
		for _, track := range tracks_delta {
			track_file := youtube.FetchAndDownload(track, *arg_music_folder)
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

func MetadataAndMove(track_file string, track Track, wg *sync.WaitGroup) {
	track_mp3, err := id3.Open(track_file)
	if err != nil {
		fmt.Println("Something bad happened while opening " + track_file + ".")
		os.Exit(1)
	} else {
		fmt.Println("Fixing metadata for:", track.Artist+" - "+track.Title)
		track_mp3.SetTitle(track.Title)
		track_mp3.SetArtist(track.Artist)
		track_mp3.SetAlbum(track.Album)
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
