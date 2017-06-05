package main

import (
	"flag"
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
	logger           = NewLogger()
)

func main() {
	arg_music_folder = flag.String("music", "~/Music", "Folder to sync with music.")
	flag.Parse()
	if !(IsDir(*arg_music_folder)) {
		logger.Fatal("Chosen music folder does not exist: " + *arg_music_folder)
	} else {
		logger.Log("Synchronization folder: " + *arg_music_folder)
	}

	wait_group.Add(1)
	go LocalLibrary(&wait_group)
	tracks_online := spotify.AuthAndTracks()
	wait_group.Wait()

	logger.Log("Checking which songs need to be downloaded.")
	for _, track_online := range tracks_online {
		track := Track{
			Title:        track_online.FullTrack.SimpleTrack.Name,
			Artist:       (track_online.FullTrack.SimpleTrack.Artists[0]).Name,
			Album:        track_online.FullTrack.Album.Name,
			Filename:     (track_online.FullTrack.SimpleTrack.Artists[0]).Name + " - " + track_online.FullTrack.SimpleTrack.Name,
			FilenameTemp: "." + (track_online.FullTrack.SimpleTrack.Artists[0]).Name + " - " + track_online.FullTrack.SimpleTrack.Name,
			FilenameExt:  DEFAULT_EXTENSION,
		}.Normalize()
		if !tracks_offline.Has(track) {
			tracks_delta = append(tracks_delta, track)
		}
	}

	if len(tracks_delta) > 0 {
		logger.Log(strconv.Itoa(len(tracks_delta)) + " missing songs, " + strconv.Itoa(len(tracks_online)-len(tracks_delta)) + " ignored.")
		for _, track := range tracks_delta {
			err := youtube.FetchAndDownload(track, *arg_music_folder)
			if err != nil {
				logger.Log("Something went wrong with \"" + track.Filename + "\": " + err.Error() + ".")
			} else {
				wait_group.Add(1)
				go MetadataAndMove(track, &wait_group)
			}
		}
		wait_group.Wait()

		logger.Log("Synchronization completed.")
	} else {
		logger.Log("No song to download.")
	}
}

func LocalLibrary(wg *sync.WaitGroup) {
	logger.Log("Reading files in local storage \"" + *arg_music_folder + "\".")
	tracks_files, _ := ioutil.ReadDir(*arg_music_folder)
	for _, track_file_info := range tracks_files {
		track_file := track_file_info.Name()
		track_file_ext := filepath.Ext(track_file)
		if track_file_ext != DEFAULT_EXTENSION {
			continue
		}
		track_file_name := track_file[0 : len(track_file)-len(track_file_ext)]
		track_title := strings.Split(track_file_name, " - ")[1]
		track_artist := strings.Split(track_file_name, " - ")[0]
		track_album := "none"
		track := Track{
			Title:        track_title,
			Artist:       track_artist,
			Album:        track_album,
			Filename:     track_file_name,
			FilenameTemp: "." + track_file_name,
			FilenameExt:  track_file_ext,
		}
		tracks_offline = append(tracks_offline, track)
	}

	wg.Done()
}

func MetadataAndMove(track Track, wg *sync.WaitGroup) {
	src_file := *arg_music_folder + "/" + track.FilenameTemp + track.FilenameExt
	dst_file := *arg_music_folder + "/" + track.Filename + track.FilenameExt
	track_mp3, err := id3.Open((*arg_music_folder) + "/" + track.FilenameTemp + track.FilenameExt)
	if err != nil {
		logger.Fatal("Something bad happened while opening " + track.Filename + ": " + err.Error() + ".")
	} else {
		logger.Log("Fixing metadata for: " + track.Filename + ".")
		track_mp3.SetTitle(track.Title)
		track_mp3.SetArtist(track.Artist)
		track_mp3.SetAlbum(track.Album)
		defer track_mp3.Close()
	}

	os.Remove(dst_file)
	err = os.Rename(src_file, dst_file)
	if err != nil {
		logger.Fatal("Something went wrong while moving song from \"" + src_file + "\" to \"" + dst_file + "\": " + err.Error() + ".")
	} else {
		logger.Log("Fixed metadata and moved song to \"" + dst_file + "\".")
	}

	wg.Done()
}
