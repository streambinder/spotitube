package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"spotify"
	"strconv"
	"strings"
	"sync"
	"youtube"

	id3 "github.com/mikkyang/id3-go"
	api "github.com/zmb3/spotify"
	. "utils"
)

var (
	tracks_offline Tracks
	tracks_delta   Tracks
	tracks_failed  Tracks
	wait_group     sync.WaitGroup
	arg_folder     *string
	arg_playlist   *string
	logger         = NewLogger()
)

func main() {
	arg_folder = flag.String("folder", "~/Music", "Folder to sync with music.")
	arg_playlist = flag.String("playlist", "none", "Playlist URI to synchronize.")
	flag.Parse()
	if !(IsDir(*arg_folder)) {
		logger.Fatal("Chosen music folder does not exist: " + *arg_folder)
	} else {
		logger.Log("Synchronization folder: " + *arg_folder)
	}

	wait_group.Add(1)
	go LocalLibrary(&wait_group)
	var tracks_online []api.FullTrack
	if *arg_playlist == "none" {
		tracks_online = spotify.AuthAndTracks()
	} else {
		tracks_online = spotify.AuthAndTracks(*arg_playlist)
	}
	wait_group.Wait()

	logger.Log("Checking which songs need to be downloaded.")
	for _, track_online := range tracks_online {
		track := Track{
			Title:        track_online.SimpleTrack.Name,
			Artist:       (track_online.SimpleTrack.Artists[0]).Name,
			Album:        track_online.Album.Name,
			Filename:     (track_online.SimpleTrack.Artists[0]).Name + " - " + track_online.SimpleTrack.Name,
			FilenameTemp: "." + (track_online.SimpleTrack.Artists[0]).Name + " - " + track_online.SimpleTrack.Name,
			FilenameExt:  DEFAULT_EXTENSION,
		}.Normalize()
		if !tracks_offline.Has(track) {
			tracks_delta = append(tracks_delta, track)
		}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		logger.Log("SIGINT captured: cleaning up temporary files.")
		for _, track := range tracks_delta {
			os.Remove(*arg_folder + "/" + track.FilenameTemp)
			os.Remove(*arg_folder + "/" + track.FilenameTemp + track.FilenameExt)
			os.Remove(*arg_folder + "/" + track.FilenameTemp + ".part")
			os.Remove(*arg_folder + "/" + track.FilenameTemp + ".part*")
			os.Remove(*arg_folder + "/" + track.FilenameTemp + ".ytdl")
		}
		logger.Fatal("Explicit closure request by the user. Exiting.")
	}()

	if len(tracks_delta) > 0 {
		logger.Log(strconv.Itoa(len(tracks_delta)) + " missing songs, " + strconv.Itoa(len(tracks_online)-len(tracks_delta)) + " ignored.")
		for track_index, track := range tracks_delta {
			logger.Log(strconv.Itoa(track_index+1) + "/" + strconv.Itoa(len(tracks_delta)) + ": " + track.Filename)
			err := youtube.FetchAndDownload(track, *arg_folder)
			if err != nil {
				logger.Log("Something went wrong with \"" + track.Filename + "\": " + err.Error() + ".")
				tracks_failed = append(tracks_failed, track)
			} else {
				wait_group.Add(1)
				go MetadataAndMove(track, &wait_group)
			}
		}
		wait_group.Wait()

		if len(tracks_failed) > 0 {
			logger.Log("Synchronization partially completed, " + strconv.Itoa(len(tracks_failed)) + " tracks failed to synchronize:")
			var tracks_failed_output string
			for track_index, track := range tracks_failed {
				if track_index == 0 {
					tracks_failed_output = "\"" + track.Filename + "\""
				} else {
					tracks_failed_output = tracks_failed_output + ", \"" + track.Filename + "\""
				}
			}
			logger.Log(tracks_failed_output + ".")
		} else {
			logger.Log("Synchronization completed.")
		}
	} else {
		logger.Log("No song to download.")
	}
}

func LocalLibrary(wg *sync.WaitGroup) {
	logger.Log("Reading files in local storage \"" + *arg_folder + "\".")
	tracks_files, _ := ioutil.ReadDir(*arg_folder)
	for _, track_file_info := range tracks_files {
		track_file := track_file_info.Name()
		track_file_ext := filepath.Ext(track_file)
		if track_file_ext != DEFAULT_EXTENSION || !strings.Contains(track_file, " - ") {
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
	src_file := *arg_folder + "/" + track.FilenameTemp + track.FilenameExt
	dst_file := *arg_folder + "/" + track.Filename + track.FilenameExt
	track_mp3, err := id3.Open((*arg_folder) + "/" + track.FilenameTemp + track.FilenameExt)
	if err != nil {
		logger.Fatal("Something bad happened while opening " + track.Filename + ": " + err.Error() + ".")
	} else {
		logger.Log("Fixing metadata for: " + track.Filename + ".")
		track_mp3.SetTitle(track.Title)
		track_mp3.SetArtist(track.Artist)
		track_mp3.SetAlbum(track.Album)
		track_mp3.Close()
	}

	os.Remove(dst_file)
	command_cmd := "ffmpeg"
	command_args := []string{"-i", src_file, "-codec:a", "libmp3lame", "-qscale:a", "0", dst_file}
	_, err = exec.Command(command_cmd, command_args...).Output()
	if err != nil {
		logger.Fatal("Something went wrong while standardizing " + track.FilenameExt[1:] + " structure via \"" + command_cmd + " " + strings.Join(command_args, " ") + "\": " + err.Error())
	} else {
		logger.Log("Fixed metadata, standardized " + track.FilenameExt[1:] + " structure and moved song to \"" + dst_file + "\".")
	}

	wg.Done()
}
