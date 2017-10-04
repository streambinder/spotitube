package main

import (
	"bufio"
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"spotify"
	"strconv"
	"strings"
	"sync"
	"youtube"

	id3 "github.com/bogem/id3v2"
	api "github.com/zmb3/spotify"
	. "utils"
)

var (
	tracks_offline            Tracks
	tracks_delta              Tracks
	tracks_failed             Tracks
	wait_group                sync.WaitGroup
	arg_folder                *string
	arg_playlist              *string
	arg_flush_metadata        *bool
	arg_disable_normalization *bool
	arg_interactive           *bool
	arg_log                   *bool
	arg_debug                 *bool
	logger                    Logger = NewLogger()
)

func main() {
	arg_folder = flag.String("folder", ".", "Folder to sync with music.")
	arg_playlist = flag.String("playlist", "none", "Playlist URI to synchronize.")
	arg_flush_metadata = flag.Bool("flush-metadata", false, "Flush metadata informations to already synchronized songs")
	arg_disable_normalization = flag.Bool("disable-normalization", false, "Disable songs volume normalization")
	arg_interactive = flag.Bool("interactive", false, "Enable interactive mode")
	arg_log = flag.Bool("log", false, "Enable logging into file ./spotify-dl.log")
	arg_debug = flag.Bool("debug", false, "Enable debug messages")
	flag.Parse()

	if *arg_log {
		logger.SetFile(DEFAULT_LOG_PATH)
	}

	if *arg_debug {
		logger.EnableDebug()
	}

	if !(IsDir(*arg_folder)) {
		logger.Fatal("Chosen music folder does not exist: " + *arg_folder)
	} else {
		logger.Log("Synchronization folder: " + *arg_folder)
	}

	var tracks_online []api.FullTrack
	if *arg_playlist == "none" {
		tracks_online = spotify.AuthAndTracks()
	} else {
		tracks_online = spotify.AuthAndTracks(*arg_playlist)
	}

	logger.Log("Checking which songs need to be downloaded.")
	for _, track_online := range tracks_online {
		track := Track{
			Title:  track_online.SimpleTrack.Name,
			Artist: (track_online.SimpleTrack.Artists[0]).Name,
			Album:  track_online.Album.Name,
			Featurings: func() []string {
				var featurings []string
				if len(track_online.SimpleTrack.Artists) > 1 {
					for _, artist_item := range track_online.SimpleTrack.Artists[1:] {
						featurings = append(featurings, artist_item.Name)
					}
				}
				return featurings
			}(),
			Image:         track_online.Album.Images[0],
			Duration:      track_online.SimpleTrack.Duration,
			Filename:      "",
			FilenameTemp:  "",
			FilenameExt:   DEFAULT_EXTENSION,
			SearchPattern: "",
		}.Normalize()
		if _, err := os.Stat(*arg_folder + "/" + track.Filename + track.FilenameExt); !os.IsNotExist(err) {
			tracks_offline = append(tracks_offline, track)
		} else {
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
		youtube.SetInteractive(arg_interactive)
		logger.Log(strconv.Itoa(len(tracks_delta)) + " missing songs, " + strconv.Itoa(len(tracks_online)-len(tracks_delta)) + " ignored.")
		for track_index, track := range tracks_delta {
			logger.Log(strconv.Itoa(track_index+1) + "/" + strconv.Itoa(len(tracks_delta)) + ": \"" + track.Filename + "\"")
			err := youtube.FetchAndDownload(track, *arg_folder)
			if err != nil {
				logger.Log("Something went wrong with \"" + track.Filename + "\": " + err.Error() + ".")
				tracks_failed = append(tracks_failed, track)
			} else {
				wait_group.Add(1)
				go MetadataNormalizeAndMove(track, &wait_group)
				if *arg_debug {
					wait_group.Wait()
				}
			}
		}
		if *arg_flush_metadata {
			for track_index, track := range tracks_offline {
				logger.Log("Local " + strconv.Itoa(track_index+1) + "/" + strconv.Itoa(len(tracks_offline)) + ": \"" + track.Filename + "\"")
				os.Remove(*arg_folder + "/" + track.FilenameTemp + track.FilenameExt)
				os.Rename(*arg_folder+"/"+track.Filename+track.FilenameExt,
					*arg_folder+"/"+track.FilenameTemp+track.FilenameExt)
				wait_group.Add(1)
				go MetadataNormalizeAndMove(track, &wait_group)
				if *arg_debug {
					wait_group.Wait()
				}
			}
		}
		wait_group.Wait()

		if len(tracks_failed) > 0 {
			logger.Log("Synchronization partially completed, " + strconv.Itoa(len(tracks_failed)) + " tracks failed to synchronize:")
			for _, track := range tracks_failed {
				logger.Log(" - \"" + track.Filename + "\"")
			}
		} else {
			logger.Log("Synchronization completed.")
		}
	} else {
		logger.Log("No song needs to be downloaded.")
	}
	if *arg_flush_metadata {
		if len(tracks_offline) > 0 {
			logger.Log("Flushing metadata for " + strconv.Itoa(len(tracks_offline)) + " songs.")
		}
		for track_index, track := range tracks_offline {
			logger.Log("Local " + strconv.Itoa(track_index+1) + "/" + strconv.Itoa(len(tracks_offline)) + ": \"" + track.Filename + "\"")
			os.Remove(*arg_folder + "/" + track.FilenameTemp + track.FilenameExt)
			os.Rename(*arg_folder+"/"+track.Filename+track.FilenameExt,
				*arg_folder+"/"+track.FilenameTemp+track.FilenameExt)
			wait_group.Add(1)
			go MetadataNormalizeAndMove(track, &wait_group)
			if *arg_debug {
				wait_group.Wait()
			}
		}
		logger.Log("Job done.")
	}
	wait_group.Wait()
}

func MetadataNormalizeAndMove(track Track, wg *sync.WaitGroup) {
	defer wg.Done()

	img_file_path := *arg_folder + "/" + track.FilenameTemp + ".jpg"
	img_file, err := os.Create(img_file_path)
	if err != nil {
		logger.Warn("Something wrong while creating artwork file: " + err.Error())
	}
	img_file_writer := bufio.NewWriter(img_file)
	err = track.Image.Download(img_file_writer)
	if err != nil {
		logger.Warn("Something wrong while downloading artwork file: " + err.Error())
	}
	defer os.Remove(img_file_path)
	defer img_file.Close()

	src_file := *arg_folder + "/" + track.FilenameTemp + track.FilenameExt
	dst_file := *arg_folder + "/" + track.Filename + track.FilenameExt
	track_mp3, err := id3.Open(src_file, id3.Options{Parse: true})
	if track_mp3 == nil || err != nil {
		logger.Fatal("Error while parsing mp3 file: " + err.Error())
	}
	defer track_mp3.Close()
	if err != nil {
		logger.Fatal("Something bad happened while opening " + track.Filename + ": " + err.Error() + ".")
	} else {
		logger.Log("Fixing metadata for: " + track.Filename + ".")
		track_mp3.SetTitle(track.Title)
		track_mp3.SetArtist(track.Artist)
		track_mp3.SetAlbum(track.Album)
		track_artwork_read, err := ioutil.ReadFile(img_file_path)
		if err != nil {
			logger.Warn("Unable to read artwork file: " + err.Error())
		}
		track_artwork := id3.PictureFrame{
			Encoding:    id3.EncodingUTF8,
			MimeType:    "image/jpeg",
			PictureType: id3.PTFrontCover,
			Description: "Front cover",
			Picture:     track_artwork_read,
		}
		track_mp3.AddAttachedPicture(track_artwork)
		track_mp3.Save()
	}

	if !(*arg_disable_normalization) {
		defer os.Remove(src_file)
		os.Remove(dst_file)

		var (
			command_cmd         = "ffmpeg"
			command_args        []string
			command_out         bytes.Buffer
			command_err         error
			normalization_delta string
		)

		command_args = []string{"-i", src_file, "-af", "volumedetect", "-f", "null", "-y", "null"}
		logger.Debug("Getting max_volume value: \"" + command_cmd + " " + strings.Join(command_args, " ") + "\".")
		command_obj := exec.Command(command_cmd, command_args...)
		command_obj.Stderr = &command_out
		command_err = command_obj.Run()
		if command_err != nil {
			logger.Warn("Unable to use ffmpeg to pull max_volume song value: " + command_err.Error() + ".")
			normalization_delta = "0"
		} else {
			command_scanner := bufio.NewScanner(strings.NewReader(command_out.String()))
			for command_scanner.Scan() {
				if strings.Contains(command_scanner.Text(), "max_volume:") {
					normalization_delta = strings.Split(strings.Split(command_scanner.Text(), "max_volume:")[1], " ")[1]
					normalization_delta = strings.Replace(normalization_delta, "-", "", -1)
				}
			}
		}

		if _, command_err = strconv.ParseFloat(normalization_delta, 64); command_err != nil {
			logger.Warn("Unable to pull max_volume delta to be applied along with song volume normalization: " + normalization_delta + ".")
			normalization_delta = "0"
		}
		command_args = []string{"-i", src_file, "-af", "volume=+" + normalization_delta + "dB", "-b:a", "320k", "-y", dst_file}
		logger.Log("Normalizing volume by " + normalization_delta + "dB for: " + track.Filename + ".")
		logger.Debug("Using command: \"" + command_cmd + " " + strings.Join(command_args, " ") + "\"")
		if _, command_err = exec.Command(command_cmd, command_args...).Output(); command_err != nil {
			logger.Warn("Something went wrong while normalizing song \"" + track.Filename + "\" volume: " + command_err.Error())
		}
	} else {
		os.Rename(src_file, dst_file)
	}
}
