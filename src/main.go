package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"./command"
	"./cui"
	"./logger"
	"./provider"
	"./spotify"
	"./spotitube"
	"./system"
	"./track"

	"github.com/bogem/id3v2"
	"github.com/gosimple/slug"
)

var (
	// flags
	argFolder                string
	argPlaylist              string // TODO: it should be an array of strings
	argFlushCache            bool
	argFlushLocal            bool
	argFlushMetadata         bool
	argDisableNormalization  bool
	argDisablePlaylistFile   bool
	argPlsFile               bool
	argDisableLyrics         bool
	argDisableTimestampFlush bool
	argDisableUpdateCheck    bool
	argDisableBrowserOpening bool
	argDisableIndexing       bool
	argAuthenticateOutside   bool
	argInteractive           bool
	argManualInput           bool
	argRemoveDuplicates      bool
	argCleanJunks            bool
	argLog                   bool
	argDisableGui            bool
	argDebug                 bool
	argSimulate              bool
	argVersion               bool
	argFix                   system.PathsArrayFlag

	// spotify
	c            *spotify.Client
	cUsr         string
	cUsrID       string
	tracks       track.Tracks
	tracksFailed track.Tracks
	tracksIndex  *track.TracksIndex
	// TODO: drop
	playlistInfo *spotify.Playlist

	// routines
	waitGroup     sync.WaitGroup
	waitGroupPool = make(chan bool, spotitube.ConcurrencyLimit)

	// cli
	ui *cui.CUI
)

func main() {
	mainSetup()
	mainFork()
	mainEnvCheck()
	mainFlagsParse()
	mainInit()
	mainUI()
	mainFetch()
	mainExit()
}

func mainSetup() {
	system.Mkdir(spotitube.UserPath())
}

func mainFork() {
	if system.Proc() != spotitube.UserBinary &&
		system.FileExists(spotitube.UserBinary) &&
		os.Getenv("SPOTITUBE_FORKED") != "1" {
		syscall.Exec(spotitube.UserBinary, os.Args, append(os.Environ(), []string{"SPOTITUBE_FORKED=1"}...))
		mainExit()
	}
}

func mainEnvCheck() {
	if len(spotify.SpotifyClientID) != 32 && len(os.Getenv("SPOTIFY_ID")) != 32 {
		fmt.Println("SPOTIFY_ID environment key is unset.")
		os.Exit(1)
	}

	if len(spotify.SpotifyClientSecret) != 32 && len(os.Getenv("SPOTIFY_KEY")) != 32 {
		fmt.Println("SPOTIFY_KEY environment key is unset.")
		os.Exit(1)
	}

	if len(track.GeniusAccessToken) != 64 && len(os.Getenv("GENIUS_TOKEN")) != 64 {
		fmt.Println("GENIUS_TOKEN environment key is unset: you won't be able to fetch lyrics from Genius.")
		time.Sleep(3 * time.Second)
	}

	if !command.YoutubeDL().Exists() {
		fmt.Println(fmt.Sprintf("%s command is not installed.", command.YoutubeDL().Name()))
		os.Exit(1)
	}

	if !command.FFmpeg().Exists() {
		fmt.Println(fmt.Sprintf("%s command is not installed.", command.FFmpeg().Name()))
		os.Exit(1)
	}

	if !system.IsOnline() {
		fmt.Println("You're not connected to the internet.")
		os.Exit(1)
	}

	if upstream, err := spotitube.UpstreamVersion(); err == nil &&
		!argDisableUpdateCheck && upstream != spotitube.Version {
		fmt.Println(fmt.Sprintf("You're not running latest version (%d). Going to update.", upstream))
		if err := spotitube.UpstreamDownload(spotitube.UserBinary); err != nil {
			fmt.Println(fmt.Sprintf("Unable to update: %s", err.Error()))
		} else {
			mainFork()
		}
	}
}

func mainFlagsParse() {
	flag.StringVar(&argFolder, "folder", ".", "Folder to sync your tracks collection into")
	flag.StringVar(&argPlaylist, "playlist", "none", "Playlist URI to synchronize")
	flag.BoolVar(&argFlushCache, "flush-cache", false, "Force Spotify tracks collection cache flush")
	flag.BoolVar(&argFlushLocal, "replace-local", false, "Flush already downloaded tracks if better results get encountered")
	flag.BoolVar(&argFlushMetadata, "flush-metadata", false, "Flush metadata tags to already synchronized songs")
	flag.BoolVar(&argDisableNormalization, "disable-normalization", false, "Disable songs volume normalization")
	flag.BoolVar(&argDisablePlaylistFile, "disable-playlist-file", false, "Disable automatic creation of playlists file")
	flag.BoolVar(&argPlsFile, "pls-file", false, "Generate playlist file with .pls instead of .m3u")
	flag.BoolVar(&argDisableLyrics, "disable-lyrics", false, "Disable download of songs lyrics and their application into mp3")
	flag.BoolVar(&argDisableTimestampFlush, "disable-timestamp-flush", false, "Disable automatic songs files timestamps flush")
	flag.BoolVar(&argDisableUpdateCheck, "disable-update-check", false, "Disable automatic update check at startup")
	flag.BoolVar(&argDisableBrowserOpening, "disable-browser-opening", false, "Disable automatic browser opening for authentication")
	flag.BoolVar(&argDisableIndexing, "disable-indexing", false, "Disable automatic library indexing (used to keep track of tracks names modifications)")
	flag.BoolVar(&argAuthenticateOutside, "authenticate-outside", false, "Enable authentication flow to be handled outside this machine")
	flag.BoolVar(&argInteractive, "interactive", false, "Enable interactive mode")
	flag.BoolVar(&argManualInput, "manual-input", false, "Always manually insert URL used for songs download")
	flag.BoolVar(&argRemoveDuplicates, "remove-duplicates", false, "Remove encountered duplicates from online library/playlist")
	flag.BoolVar(&argCleanJunks, "clean-junks", false, "Scan for junks file and clean them")
	flag.BoolVar(&argLog, "log", false, "Enable logging into file ./spotitube.log")
	flag.BoolVar(&argDebug, "debug", false, "Enable debug messages")
	flag.BoolVar(&argSimulate, "simulate", false, "Simulate process flow, without really altering filesystem")
	flag.BoolVar(&argVersion, "version", false, "Print version")
	flag.Var(&argFix, "fix", "Offline song filename(s) which straighten the shot to")
	// TODO: gocui should work on windows, too
	if runtime.GOOS != "windows" {
		flag.BoolVar(&argDisableGui, "disable-gui", false, "Disable GUI to reduce noise and increase readability of program flow")
	} else {
		argDisableGui = true
	}
	flag.Parse()

	if len(argFix.Paths) > 0 {
		argFlushLocal = true
		argFlushMetadata = true
	}

	if argManualInput {
		argInteractive = true
	}
}

func mainInit() {
	system.TrapSignal(os.Interrupt, func() {
		mainExit()
	})

	if argVersion {
		fmt.Println(fmt.Sprintf("SpotiTube, version %d.", spotitube.Version))
		os.Exit(0)
	}

	if argCleanJunks {
		fmt.Println(fmt.Sprintf("Removed %d junks.", system.FileWildcardDelete(argFolder, track.JunkWildcards()...)))
		os.Exit(0)
	}

	if !(system.Dir(argFolder)) {
		fmt.Println(fmt.Sprintf("Chosen music folder does not exist: %s", argFolder))
		os.Exit(1)
	}
	os.Chdir(argFolder)

	if !argDisableIndexing {
		if system.FileExists(spotitube.UserIndex) {
			system.FetchGob(spotitube.UserIndex, tracksIndex)
		}

		tracksIndex = track.Index(argFolder)
	}

}

func mainUI() {
	var (
		uiOpts uint64
		uiErr  error
	)

	if argDebug {
		uiOpts |= cui.GuiDebugMode
	}

	if argDisableGui {
		uiOpts |= cui.GuiBareMode
	}

	if argLog {
		uiOpts |= cui.LogEnable
	}

	if ui, uiErr = cui.Startup(uiOpts); uiErr != nil {
		fmt.Println(fmt.Sprintf("Unable to build user interface: %s", uiErr.Error()))
		os.Exit(1)
	}

	ui.OnShutdown(func() {
		mainExit()
	})

	ui.Append(fmt.Sprintf("%s %s", cui.Font("Folder:", cui.StyleBold), system.PrettyPath(argFolder)), cui.PanelLeftTop)
	ui.Append(fmt.Sprintf("%s %s", cui.Font(fmt.Sprintf("%s version:", command.YoutubeDL().Name()), cui.StyleBold), command.YoutubeDL().Version()), cui.PanelLeftTop)
	ui.Append(fmt.Sprintf("%s %s", cui.Font(fmt.Sprintf("%s version:", command.FFmpeg().Name()), cui.StyleBold), command.FFmpeg().Version()), cui.PanelLeftTop)
	ui.Append(fmt.Sprintf("%s %d", cui.Font("Version:", cui.StyleBold), spotitube.Version), cui.PanelLeftBottom)
	ui.Append(fmt.Sprintf("%s %s", cui.Font("Proc:", cui.StyleBold), system.PrettyPath(system.Proc())), cui.PanelLeftBottom)
	ui.Append(fmt.Sprintf("%s %s", cui.Font("Date:", cui.StyleBold), time.Now().Format("2006-01-02 15:04:05")), cui.PanelLeftBottom)
	ui.Append(fmt.Sprintf("%s %s", cui.Font("URL:", cui.StyleBold), spotitube.RepositoryURI), cui.PanelLeftBottom)
	ui.Append(fmt.Sprintf("%s GPL-3.0", cui.Font("License:", cui.StyleBold)), cui.PanelLeftBottom)

	if argLog {
		ui.Append(fmt.Sprintf("%s %s", cui.Font("Log:", cui.StyleBold), logger.LogFilename), cui.PanelLeftTop)
	}
}

func mainFetch() {
	if len(argFix.Paths) == 0 {
		var spotifyAuthHost string
		if !argAuthenticateOutside {
			spotifyAuthHost = "localhost"
		} else {
			argDisableBrowserOpening = true
			spotifyAuthHost = "spotitube.local"
			ui.Prompt("Outside authentication enabled: assure \"spotitube.local\" points to this machine.")
		}
		spotifyAuthURL := spotify.BuildAuthURL(spotifyAuthHost)
		ui.Append(fmt.Sprintf("Authentication URL: %s", spotifyAuthURL.Short), cui.ParagraphAutoReturn)
		if !argDisableBrowserOpening {
			ui.Append("Waiting for automatic login process. If wait is too long, manually open that URL.", cui.DebugAppend)
		}
		var err error
		if c, err = spotify.Auth(spotifyAuthURL.Full, spotifyAuthHost, !argDisableBrowserOpening); err != nil {
			ui.Prompt("Unable to authenticate to spotify.", cui.PromptExit)
		}
		ui.Append("Authentication completed.")
		cUsr, cUsrID = c.User()
		ui.Append(fmt.Sprintf("%s %s", cui.Font("Session user:", cui.StyleBold), cUsr), cui.PanelLeftTop)

		var (
			tracksOnline          []spotify.Track
			tracksOnlineAlbums    []spotify.Album
			tracksOnlineAlbumsIds []spotify.ID
			tracksErr             error
			gob                   string
		)

		if argPlaylist == "none" {
			gob = fmt.Sprintf(spotitube.UserGob, cUsrID, "library")
			if argFlushCache {
				os.Remove(gob)
			}
			tracksDump, tracksDumpErr := subFetchGob(gob)
			if tracksDumpErr != nil {
				ui.Append(tracksDumpErr.Error(), cui.WarningAppend)
				ui.Append("Fetching music library...")
				if tracksOnline, tracksErr = c.LibraryTracks(); tracksErr != nil {
					ui.Prompt(fmt.Sprintf("Something went wrong while fetching tracks from library: %s.", tracksErr.Error()), cui.PromptExit)
				}
			} else {
				ui.Append(fmt.Sprintf("Tracks loaded from cache."))
				ui.Append(fmt.Sprintf("%s %d/%d (min)", cui.Font("Tracks cache lifetime:", cui.StyleBold), int(time.Since(tracksDump.Time).Minutes()), 30), cui.PanelLeftTop)
				for _, t := range tracksDump.Tracks {
					tracks = append(tracks, t.FlushLocal())
				}
			}
		} else {
			ui.Append("Fetching playlist data...")
			var playlistErr error
			playlistInfo, playlistErr = c.Playlist(argPlaylist)
			if playlistErr != nil {
				ui.Prompt("Something went wrong while fetching playlist info.", cui.PromptExit)
			} else {
				ui.Append(fmt.Sprintf("%s %s", cui.Font("Playlist name:", cui.StyleBold), playlistInfo.Name), cui.PanelLeftTop)
				if len(playlistInfo.Owner.DisplayName) == 0 && len(strings.Split(argPlaylist, ":")) >= 3 {
					ui.Append(fmt.Sprintf("%s %s", cui.Font("Playlist owner:", cui.StyleBold), strings.Split(argPlaylist, ":")[2]), cui.PanelLeftTop)
				} else {
					ui.Append(fmt.Sprintf("%s %s", cui.Font("Playlist owner:", cui.StyleBold), playlistInfo.Owner.DisplayName), cui.PanelLeftTop)
				}

				gob = fmt.Sprintf(spotitube.UserGob, playlistInfo.Owner.ID, playlistInfo.Name)
				if argFlushCache {
					os.Remove(gob)
				}
				tracksDump, tracksDumpErr := subFetchGob(gob)
				if tracksDumpErr != nil {
					ui.Append(tracksDumpErr.Error(), cui.WarningAppend)
					ui.Append(fmt.Sprintf("Getting songs from \"%s\" playlist, by \"%s\"...", playlistInfo.Name, playlistInfo.Owner.DisplayName), cui.StyleBold)
					if tracksOnline, tracksErr = c.PlaylistTracks(argPlaylist); tracksErr != nil {
						ui.Prompt(fmt.Sprintf("Something went wrong while fetching playlist: %s.", tracksErr.Error()), cui.PromptExit)
					}
				} else {
					ui.Append(fmt.Sprintf("Tracks loaded from cache."))
					ui.Append(fmt.Sprintf("%s %d/%d (min)", cui.Font("Tracks cache lifetime:", cui.StyleBold), int(time.Since(tracksDump.Time).Minutes()), 30), cui.PanelLeftTop)
					for _, t := range tracksDump.Tracks {
						tracks = append(tracks, t.FlushLocal())
					}
				}
			}
		}
		for _, t := range tracksOnline {
			tracksOnlineAlbumsIds = append(tracksOnlineAlbumsIds, t.Album.ID)
		}
		if tracksOnlineAlbums, tracksErr = c.Albums(tracksOnlineAlbumsIds); tracksErr != nil {
			ui.Prompt(fmt.Sprintf("Something went wrong while fetching album info: %s.", tracksErr.Error()), cui.PromptExit)
		}

		ui.Append("Checking which songs need to be downloaded...")
		var (
			tracksDuplicates []spotify.ID
			tracksMap        = make(map[string]float64)
		)
		for trackIndex := len(tracksOnline) - 1; trackIndex >= 0; trackIndex-- {
			trackID := tracksOnline[trackIndex].SimpleTrack.ID
			if _, alreadyParsed := tracksMap[trackID.String()]; !alreadyParsed {
				tracks = append(tracks, track.ParseSpotifyTrack(tracksOnline[trackIndex], tracksOnlineAlbums[trackIndex]))
				tracksMap[trackID.String()] = 1
			} else {
				ui.Append(fmt.Sprintf("Ignored song duplicate \"%s\" by \"%s\".", tracksOnline[trackIndex].SimpleTrack.Name, tracksOnline[trackIndex].SimpleTrack.Artists[0].Name), cui.WarningAppend)
				tracksDuplicates = append(tracksDuplicates, trackID)
			}
		}

		if argRemoveDuplicates && len(tracksDuplicates) > 0 {
			if argPlaylist == "none" {
				if removeErr := c.RemoveLibraryTracks(tracksDuplicates); removeErr != nil {
					ui.Prompt(fmt.Sprintf("Something went wrong while removing %d duplicates: %s.", len(tracksDuplicates), removeErr.Error()))
				} else {
					ui.Append(fmt.Sprintf("%d duplicate tracks correctly removed from library.", len(tracksDuplicates)))
				}
			} else {
				if removeErr := c.RemovePlaylistTracks(argPlaylist, tracksDuplicates); removeErr != nil {
					ui.Prompt(fmt.Sprintf("Something went wrong while removing %d duplicates: %s.", len(tracksDuplicates), removeErr.Error()))
				} else {
					ui.Append(fmt.Sprintf("%d duplicate tracks correctly removed from playlist.", len(tracksDuplicates)))
				}
			}
		}

		if dumpErr := system.DumpGob(gob, track.TracksDump{Tracks: tracks, Time: time.Now()}); dumpErr != nil {
			ui.Append(fmt.Sprintf("Unable to cache tracks: %s", dumpErr.Error()), cui.WarningAppend)
		}

		ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs duplicates:", cui.StyleBold), len(tracksDuplicates)), cui.PanelLeftTop)
		ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs online:", cui.StyleBold), len(tracks)), cui.PanelLeftTop)
		ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs offline:", cui.StyleBold), tracks.CountOffline()), cui.PanelLeftTop)
		ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs missing:", cui.StyleBold), tracks.CountOnline()), cui.PanelLeftTop)

		track.IndexWait()
	} else {
		ui.Append(fmt.Sprintf("%s %d", cui.Font("Fix song(s):", cui.StyleBold), len(argFix.Paths)), cui.PanelLeftTop)
		for _, fixTrack := range argFix.Paths {
			if t, trackErr := track.OpenLocalTrack(fixTrack); trackErr != nil {
				ui.Prompt(fmt.Sprintf("Something went wrong: %s.", trackErr.Error()), cui.PromptExit)
				mainExit()
			} else {
				ui.Append(fmt.Sprintf("%+v\n", t), cui.DebugAppend)
				tracks = append(tracks, t)
			}
		}
	}

	for range [spotitube.ConcurrencyLimit]int{} {
		waitGroupPool <- true
	}

	if len(tracks) > 0 {
		mainSearch()
	} else {
		ui.Prompt("No song needs to be downloaded.", cui.PromptExit)
		mainExit()
	}
}

func mainSearch() {
	defer mainExit()

	ui.ProgressMax = len(tracks)

	songsFetch, songsFlush, songsIgnore := subCountSongs()
	ui.Append(fmt.Sprintf("%d will be downloaded, %d flushed and %d ignored", songsFetch, songsFlush, songsIgnore))

	for trackIndex, t := range tracks {
		ui.ProgressHalfIncrease()
		ui.Append(fmt.Sprintf("%d/%d: \"%s\"", trackIndex+1, len(tracks), t.Basename()), cui.StyleBold)

		if trackPath, ok := tracksIndex.Tracks[t.SpotifyID]; ok {
			if trackPath != t.Filename() {
				ui.Append(fmt.Sprintf("Track %s has been renamed: moving local one to %s", t.SpotifyID, t.Filename()))
				if err := subTrackRename(&t); err != nil {
					ui.Append(fmt.Sprintf("Unable to rename: %s", err.Error()), cui.ErrorAppend)
				}
			}
		}

		if subIfSongSearch(t) {
			var (
				entry    = new(provider.Entry)
				prov     provider.Provider
				provName string
			)
			for provName, prov = range provider.Providers {
				var (
					provEntries   = []*provider.Entry{}
					provErr       error
					entryPickAuto bool
					entryPick     bool
				)

				ui.Append(fmt.Sprintf("Searching entries on %s provider", provName))
				if !argManualInput {
					provEntries, provErr = prov.Query(&t)
					if provErr != nil {
						ui.Append(fmt.Sprintf("Something went wrong while searching for \"%s\" track: %s.", t.Basename(), provErr.Error()), cui.WarningAppend)
						tracksFailed = append(tracksFailed, t)
						ui.ProgressHalfIncrease()
						continue
					}
					for _, provEntry := range provEntries {
						ui.Append(fmt.Sprintf("Result met: ID: %s,\nTitle: %s,\nUser: %s,\nDuration: %d.",
							provEntry.ID, provEntry.Title, provEntry.User, provEntry.Duration), cui.DebugAppend)

						entryPickAuto, entryPick = subMatchResult(prov, &t, provEntry)
						if subIfPickFromAns(entryPickAuto, entryPick) {
							ui.Append(fmt.Sprintf("Video \"%s\" is good to go for \"%s\".", provEntry.Title, t.Basename()))
							entry = provEntry
							break
						}
					}
				}

				subCondManualInputURL(prov, entry, &t)
				if entry.URL == "" {
					entry = &provider.Entry{}
				}

				if entry == (&provider.Entry{}) {
					ui.Append(fmt.Sprintf("Video for \"%s\" not found.", t.Basename()), cui.ErrorAppend)
					tracksFailed = append(tracksFailed, t)
					ui.ProgressHalfIncrease()
					continue
				}

				if argSimulate {
					ui.Append(fmt.Sprintf("I would like to download \"%s\" for \"%s\" track, but I'm just simulating.", entry.URL, t.Basename()))
					ui.ProgressHalfIncrease()
					continue
				} else if argFlushLocal {
					if t.URL == entry.URL && !entryPick {
						ui.Append(fmt.Sprintf("Track \"%s\" is still the best result I can find.", t.Basename()))
						ui.Append(fmt.Sprintf("Local track origin URL %s is the same as the chosen one %s.", t.URL, entry.URL), cui.DebugAppend)
						ui.ProgressHalfIncrease()
						continue
					} else {
						t.URL = ""
						t.Local = false
					}
				}
			}

			ui.Append(fmt.Sprintf("Going to download \"%s\" from %s...", entry.Title, entry.URL))
			err := prov.Download(entry, t.FilenameTemporary())
			if err != nil {
				ui.Append(fmt.Sprintf("Something went wrong downloading \"%s\": %s.", t.Basename(), err.Error()), cui.WarningAppend)
				tracksFailed = append(tracksFailed, t)
				ui.ProgressHalfIncrease()
				continue
			} else {
				t.URL = entry.URL
			}
		}

		if !subIfSongProcess(t) {
			ui.ProgressHalfIncrease()
			continue
		}

		subCondSequentialDo(&t)

		ui.Append(fmt.Sprintf("Launching song processing jobs..."))
		waitGroup.Add(1)
		go subParallelSongProcess(t, &waitGroup)
		if argDebug {
			waitGroup.Wait()
		}
	}
	waitGroup.Wait()

	subCondPlaylistFileWrite()
	subCondTimestampFlush()
	subWriteIndex()

	close(waitGroupPool)
	waitGroup.Wait()
	ui.ProgressFill()

	ui.Append(fmt.Sprintf("%d tracks failed to synchronize.", len(tracksFailed)))
	for _, t := range tracksFailed {
		ui.Append(fmt.Sprintf(" - \"%s\"", t.Basename()))
	}

	system.Notify("SpotiTube", "emblem-downloads", "SpotiTube", fmt.Sprintf("%d track(s) synced, %d failed.", len(tracks)-len(tracksFailed), len(tracksFailed)))
	ui.Prompt("Synchronization completed.", cui.PromptExit)
}

func mainExit(delay ...time.Duration) {
	system.FileWildcardDelete(argFolder, track.JunkWildcards()...)

	if len(delay) > 0 {
		time.Sleep(delay[0])
	}

	os.Exit(0)
}

func subWriteIndex() {
	ui.Append(fmt.Sprintf("Writing %d entries index...", len(tracksIndex.Tracks)), cui.DebugAppend)
	if writeErr := system.DumpGob(spotitube.UserIndex, tracksIndex); writeErr != nil {
		ui.Append(fmt.Sprintf("Unable to write tracks index: %s", writeErr.Error()), cui.WarningAppend)
	}
}

func subParallelSongProcess(t track.Track, wg *sync.WaitGroup) {
	defer ui.ProgressHalfIncrease()
	defer wg.Done()
	<-waitGroupPool

	if !t.Local && !argDisableNormalization {
		subSongNormalize(t)
	}

	if !system.FileExists(t.FilenameTemporary()) && system.FileExists(t.Filename()) {
		if err := system.FileCopy(t.Filename(), t.FilenameTemporary()); err != nil {
			ui.Append(fmt.Sprintf("Unable to prepare song for getting its metadata flushed: %s", err.Error()), cui.WarningAppend)
			return
		}
	}

	if (t.Local && argFlushMetadata) || !t.Local {
		subSongFlushMetadata(t)
	}

	os.Remove(t.Filename())
	err := os.Rename(t.FilenameTemporary(), t.Filename())
	if err != nil {
		ui.Append(fmt.Sprintf("Unable to move song to its final path: %s", err.Error()), cui.WarningAppend)
	}

	waitGroupPool <- true
}

func subSongNormalize(t track.Track) {
	var (
		commandCmd         = "ffmpeg"
		commandArgs        []string
		commandOut         bytes.Buffer
		commandErr         error
		normalizationDelta string
		normalizationFile  = strings.Replace(t.FilenameTemporary(), spotitube.SongExtension, fmt.Sprintf("norm.%s", spotitube.SongExtension), -1)
	)

	commandArgs = []string{"-i", t.FilenameTemporary(), "-af", "volumedetect", "-f", "null", "-y", "null"}
	ui.Append(fmt.Sprintf("Getting max_volume value: \"%s %s\"...", commandCmd, strings.Join(commandArgs, " ")), cui.DebugAppend)
	commandObj := exec.Command(commandCmd, commandArgs...)
	commandObj.Stderr = &commandOut
	commandErr = commandObj.Run()
	if commandErr != nil {
		ui.Append(fmt.Sprintf("Unable to use ffmpeg to pull max_volume song value: %s.", commandOut.String()), cui.WarningAppend)
		normalizationDelta = "0.0"
	} else {
		commandScanner := bufio.NewScanner(strings.NewReader(commandOut.String()))
		for commandScanner.Scan() {
			if strings.Contains(commandScanner.Text(), "max_volume:") {
				normalizationDelta = strings.Split(strings.Split(commandScanner.Text(), "max_volume:")[1], " ")[1]
				normalizationDelta = strings.Replace(normalizationDelta, "-", "", -1)
			}
		}
	}

	if _, commandErr = strconv.ParseFloat(normalizationDelta, 64); commandErr != nil {
		ui.Append(fmt.Sprintf("Unable to pull max_volume delta to be applied along with song volume normalization: %s.", normalizationDelta), cui.WarningAppend)
		normalizationDelta = "0.0"
	}
	commandArgs = []string{"-i", t.FilenameTemporary(), "-af", "volume=+" + normalizationDelta + "dB", "-b:a", "320k", "-y", normalizationFile}
	ui.Append(fmt.Sprintf("Compensating volume by %sdB...", normalizationDelta), cui.DebugAppend)
	ui.Append(fmt.Sprintf("Increasing audio quality for: %s...", t.Basename()), cui.DebugAppend)
	ui.Append(fmt.Sprintf("Firing command: \"%s %s\"...", commandCmd, strings.Join(commandArgs, " ")), cui.DebugAppend)
	if _, commandErr = exec.Command(commandCmd, commandArgs...).Output(); commandErr != nil {
		ui.Append(fmt.Sprintf("Something went wrong while normalizing song \"%s\" volume: %s", t.Basename(), commandErr.Error()), cui.WarningAppend)
	}
	os.Remove(t.FilenameTemporary())
	os.Rename(normalizationFile, t.FilenameTemporary())
}

func subSongFlushMetadata(t track.Track) {
	trackMp3, err := id3v2.Open(t.FilenameTemporary(), id3v2.Options{Parse: true})
	if err != nil {
		ui.Append(fmt.Sprintf("Something bad happened while opening: %s", err.Error()), cui.WarningAppend)
	} else {
		ui.Append(fmt.Sprintf("Fixing metadata for \"%s\"...", t.Basename()), cui.DebugAppend)
		subCondFlushID3FrameTitle(t, trackMp3)
		subCondFlushID3FrameSong(t, trackMp3)
		subCondFlushID3FrameArtist(t, trackMp3)
		subCondFlushID3FrameAlbum(t, trackMp3)
		subCondFlushID3FrameGenre(t, trackMp3)
		subCondFlushID3FrameYear(t, trackMp3)
		subCondFlushID3FrameFeaturings(t, trackMp3)
		subCondFlushID3FrameTrackNumber(t, trackMp3)
		subCondFlushID3FrameTrackTotals(t, trackMp3)
		subCondFlushID3FrameArtwork(t, trackMp3)
		subCondFlushID3FrameArtworkURL(t, trackMp3)
		subCondFlushID3FrameOrigin(t, trackMp3)
		subCondFlushID3FrameDuration(t, trackMp3)
		subCondFlushID3FrameSpotifyID(t, trackMp3)
		subCondFlushID3FrameLyrics(t, trackMp3)
		trackMp3.Save()
	}
	trackMp3.Close()
}

func subCondFlushID3FrameTitle(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Title) > 0 && track.TagGetFrame(trackMp3, track.ID3FrameSong) != track.TagGetFrame(trackMp3, track.ID3FrameTitle) {
		ui.Append("Inflating title metadata...", cui.DebugAppend)
		trackMp3.SetTitle(t.Title)
	}
}

func subCondFlushID3FrameSong(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Song) > 0 && t.Song != track.TagGetFrame(trackMp3, track.ID3FrameSong) {
		ui.Append("Inflating song metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "song",
			Text:        t.Song,
		})
	}
}

func subCondFlushID3FrameArtist(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Artist) > 0 && t.Artist != track.TagGetFrame(trackMp3, track.ID3FrameArtist) {
		ui.Append("Inflating artist metadata...", cui.DebugAppend)
		trackMp3.SetArtist(t.Artist)
	}
}

func subCondFlushID3FrameAlbum(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Album) > 0 && t.Album != track.TagGetFrame(trackMp3, track.ID3FrameAlbum) {
		ui.Append("Inflating album metadata...", cui.DebugAppend)
		trackMp3.SetAlbum(t.Album)
	}
}

func subCondFlushID3FrameGenre(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Genre) > 0 && t.Genre != track.TagGetFrame(trackMp3, track.ID3FrameGenre) {
		ui.Append("Inflating genre metadata...", cui.DebugAppend)
		trackMp3.SetGenre(t.Genre)
	}
}

func subCondFlushID3FrameYear(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Year) > 0 && t.Year != track.TagGetFrame(trackMp3, track.ID3FrameYear) {
		ui.Append("Inflating year metadata...", cui.DebugAppend)
		trackMp3.SetYear(t.Year)
	}
}

func subCondFlushID3FrameFeaturings(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Featurings) > 0 && strings.Join(t.Featurings, "|") != track.TagGetFrame(trackMp3, track.ID3FrameFeaturings) {
		ui.Append("Inflating featurings metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "featurings",
			Text:        strings.Join(t.Featurings, "|"),
		})
	}
}

func subCondFlushID3FrameTrackNumber(t track.Track, trackMp3 *id3v2.Tag) {
	if t.TrackNumber > 0 && fmt.Sprintf("%d", t.TrackNumber) != track.TagGetFrame(trackMp3, track.ID3FrameTrackNumber) {
		ui.Append("Inflating track number metadata...", cui.DebugAppend)
		trackMp3.AddFrame(trackMp3.CommonID("Track number/Position in set"),
			id3v2.TextFrame{
				Encoding: id3v2.EncodingUTF8,
				Text:     strconv.Itoa(t.TrackNumber),
			})
	}
}

func subCondFlushID3FrameTrackTotals(t track.Track, trackMp3 *id3v2.Tag) {
	if t.TrackTotals > 0 && fmt.Sprintf("%d", t.TrackTotals) != track.TagGetFrame(trackMp3, track.ID3FrameTrackTotals) {
		ui.Append("Inflating total tracks number metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "trackTotals",
			Text:        fmt.Sprintf("%d", t.TrackTotals),
		})
	}
}

func subCondFlushID3FrameArtwork(t track.Track, trackMp3 *id3v2.Tag) {
	if system.FileExists(t.FilenameArtwork()) && t.Image != track.TagGetFrame(trackMp3, track.ID3FrameArtworkURL) {
		trackArtworkReader, trackArtworkErr := ioutil.ReadFile(t.FilenameArtwork())
		if trackArtworkErr != nil {
			ui.Append(fmt.Sprintf("Unable to read artwork file: %s", trackArtworkErr.Error()), cui.WarningAppend)
		} else {
			ui.Append("Inflating artwork metadata...", cui.DebugAppend)
			trackMp3.AddAttachedPicture(id3v2.PictureFrame{
				Encoding:    id3v2.EncodingUTF8,
				MimeType:    "image/jpeg",
				PictureType: id3v2.PTFrontCover,
				Description: "Front cover",
				Picture:     trackArtworkReader,
			})
		}
	}
}

func subCondFlushID3FrameArtworkURL(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Image) > 0 && t.Image != track.TagGetFrame(trackMp3, track.ID3FrameArtworkURL) {
		ui.Append("Inflating artwork url metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "artwork",
			Text:        t.Image,
		})
	}
}

func subCondFlushID3FrameOrigin(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.URL) > 0 && t.URL != track.TagGetFrame(trackMp3, track.ID3FrameOrigin) {
		ui.Append("Inflating origin url metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "origin",
			Text:        t.URL,
		})
	}
}

func subCondFlushID3FrameDuration(t track.Track, trackMp3 *id3v2.Tag) {
	if t.Duration > 0 && fmt.Sprintf("%d", t.Duration) != track.TagGetFrame(trackMp3, track.ID3FrameDuration) {
		ui.Append("Inflating duration metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "duration",
			Text:        fmt.Sprintf("%d", t.Duration),
		})
	}
}

func subCondFlushID3FrameSpotifyID(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.SpotifyID) > 0 && t.SpotifyID != track.TagGetFrame(trackMp3, track.ID3FrameSpotifyID) {
		ui.Append("Inflating Spotify ID metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "spotifyid",
			Text:        t.SpotifyID,
		})
	}
}

func subCondFlushID3FrameLyrics(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Lyrics) > 0 && !argDisableLyrics && t.Lyrics != track.TagGetFrame(trackMp3, track.ID3FrameLyrics) {
		ui.Append("Inflating lyrics metadata...", cui.DebugAppend)
		trackMp3.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
			Encoding:          id3v2.EncodingUTF8,
			Language:          "eng",
			ContentDescriptor: t.Title,
			Lyrics:            t.Lyrics,
		})
	}
}

func subIfSongSearch(t track.Track) bool {
	return !t.Local || argFlushLocal || argSimulate
}

func subFetchGob(path string) (track.TracksDump, error) {
	var tracksDump = new(track.TracksDump)
	if fetchErr := system.FetchGob(path, tracksDump); fetchErr != nil {
		return track.TracksDump{}, fmt.Errorf(fmt.Sprintf("Unable to load tracks cache: %s", fetchErr.Error()))
	}

	if time.Since(tracksDump.Time).Minutes() > 30 {
		return track.TracksDump{}, fmt.Errorf("Tracks cache declared obsolete: flushing it from Spotify")
	}

	return *tracksDump, nil
}

func subCountSongs() (int, int, int) {
	var (
		songsFetch  int
		songsFlush  int
		songsIgnore int
	)
	if argFlushLocal {
		songsFetch = len(tracks)
		songsFlush = songsFetch
	} else if argFlushMetadata {
		songsFetch = tracks.CountOnline()
		songsFlush = len(tracks)
	} else {
		songsFetch = tracks.CountOnline()
		songsFlush = songsFetch
		songsIgnore = tracks.CountOffline()
	}
	return songsFetch, songsFlush, songsIgnore
}

func subMatchResult(p provider.Provider, t *track.Track, e *provider.Entry) (bool, bool) {
	var (
		ansInput     bool
		ansAutomated bool
		ansErr       error
	)
	ansErr = p.Match(e, t)
	ansAutomated = bool(ansErr == nil)
	if argInteractive {
		ansInput = ui.Prompt(fmt.Sprintf("Do you want to download the following video for \"%s\"?\n"+
			"ID: %s\nTitle: %s\nUser: %s\nDuration: %d\nURL: %s\nResult is matching: %s",
			t.Basename(), e.ID, e.Title, e.User, e.Duration, e.URL, strconv.FormatBool(ansAutomated)), cui.PromptBinary)
	}
	return ansAutomated, ansInput
}

func subIfPickFromAns(ansAutomated bool, ansInput bool) bool {
	return (!argInteractive && ansAutomated) || (argInteractive && ansInput)
}

func subCondManualInputURL(p provider.Provider, e *provider.Entry, t *track.Track) {
	if argInteractive && e.URL == "" {
		inputURL := ui.PromptInputMessage(fmt.Sprintf("Please, manually enter URL for \"%s\"", t.Basename()), cui.PromptInput)
		if len(inputURL) > 0 {
			if err := p.ValidateURL(inputURL); err == nil {
				e.Title = "input video"
				e.URL = inputURL
			} else {
				ui.Prompt(fmt.Sprintf("Something went wrong: %s", err.Error()))
			}
		}
	}
}

func subIfSongProcess(t track.Track) bool {
	return !t.Local || argFlushMetadata || argFlushLocal
}

func subCondSequentialDo(t *track.Track) {
	if (t.Local && argFlushMetadata) || !t.Local {
		subCondLyricsFetch(t)
		subCondArtworkDownload(t)
	}
}

func subCondLyricsFetch(t *track.Track) {
	if argDisableLyrics {
		return
	}

	ui.Append(fmt.Sprintf("Fetching song \"%s\" lyrics...", t.Basename()), cui.DebugAppend)
	lyricsErr := t.SearchLyrics()
	if lyricsErr != nil {
		ui.Append(fmt.Sprintf("Something went wrong while searching for song lyrics: %s", lyricsErr.Error()), cui.WarningAppend)
	} else {
		ui.Append(fmt.Sprintf("Song lyrics found."), cui.DebugAppend)
	}
}

func subCondArtworkDownload(t *track.Track) {
	if len(t.Image) == 0 || system.FileExists(t.FilenameArtwork()) {
		return
	}

	ui.Append(fmt.Sprintf("Downloading song \"%s\" artwork at %s...", t.Basename(), t.Image), cui.DebugAppend)
	var commandOut bytes.Buffer
	commandCmd := "ffmpeg"
	commandArgs := []string{"-i", t.Image, "-q:v", "1", "-n", t.FilenameArtwork()}
	commandObj := exec.Command(commandCmd, commandArgs...)
	commandObj.Stderr = &commandOut
	if err := commandObj.Run(); err != nil {
		ui.Append(fmt.Sprintf("Unable to download artwork file \"%s\": %s", t.Image, commandOut.String()), cui.WarningAppend)
	}
}

func subCondTimestampFlush() {
	if !argDisableTimestampFlush {
		ui.Append("Flushing files timestamps...")
		now := time.Now().Local().Add(time.Duration(-1*len(tracks)) * time.Minute)
		for _, t := range tracks {
			if !system.FileExists(t.Filename()) {
				continue
			}
			if err := os.Chtimes(t.Filename(), now, now); err != nil {
				ui.Append(fmt.Sprintf("Unable to flush timestamp on %s", t.Filename()), cui.WarningAppend)
			}
			now = now.Add(1 * time.Minute)
		}
	}
}

func subCondPlaylistFileWrite() {
	if !argSimulate && !argDisablePlaylistFile && argPlaylist != "none" {
		var (
			playlistFolder  = slug.Make(playlistInfo.Name)
			playlistFname   = fmt.Sprintf("%s/%s", playlistFolder, playlistInfo.Name)
			playlistContent string
		)

		if !argPlsFile {
			playlistFname = playlistFname + ".m3u"
		} else {
			playlistFname = playlistFname + ".pls"
		}

		os.RemoveAll(playlistFolder)
		os.Mkdir(playlistFolder, 0775)
		os.Chdir(playlistFolder)
		for _, t := range tracks {
			if system.FileExists("../" + t.Filename()) {
				if err := os.Symlink("../"+t.Filename(), t.Filename()); err != nil {
					ui.Append(fmt.Sprintf("Unable to create symlink for \"%s\" in %s: %s", t.Filename(), playlistFolder, err.Error()), cui.ErrorAppend)
				}
			}
		}
		os.Chdir("..")

		ui.Append(fmt.Sprintf("Creating playlist file at %s...", playlistFname))
		if system.FileExists(playlistFname) {
			os.Remove(playlistFname)
		}

		if !argPlsFile {
			playlistContent = "#EXTM3U\n"
			for trackIndex := len(tracks) - 1; trackIndex >= 0; trackIndex-- {
				t := tracks[trackIndex]
				if system.FileExists(t.Filename()) {
					playlistContent += "#EXTINF:" + strconv.Itoa(t.Duration) + "," + t.Filename() + "\n" +
						"./" + t.Filename() + "\n"
				}
			}
		} else {
			ui.Append("Creating playlist PLS file...")
			if system.FileExists(playlistInfo.Name + ".pls") {
				os.Remove(playlistInfo.Name + ".pls")
			}
			playlistContent = "[" + playlistInfo.Name + "]\n"
			for trackIndex := len(tracks) - 1; trackIndex >= 0; trackIndex-- {
				t := tracks[trackIndex]
				trackInvertedIndex := len(tracks) - trackIndex
				if system.FileExists(t.Filename()) {
					playlistContent += "File" + strconv.Itoa(trackInvertedIndex) + "=./" + t.Filename() + "\n" +
						"Title" + strconv.Itoa(trackInvertedIndex) + "=" + t.Filename() + "\n" +
						"Length" + strconv.Itoa(trackInvertedIndex) + "=" + strconv.Itoa(t.Duration) + "\n\n"
				}
			}
			playlistContent += "NumberOfEntries=" + strconv.Itoa(len(tracks)) + "\n"
		}

		playlistFile, playlistErr := os.Create(playlistFname)
		if playlistErr != nil {
			ui.Append(fmt.Sprintf("Unable to create M3U file: %s", playlistErr.Error()), cui.WarningAppend)
		} else {
			defer playlistFile.Close()
			_, playlistErr := playlistFile.WriteString(playlistContent)
			playlistFile.Sync()
			if playlistErr != nil {
				ui.Append(fmt.Sprintf("Unable to write M3U file: %s", playlistErr.Error()), cui.WarningAppend)
			}
		}
	}
}

func subTrackRename(t *track.Track) error {
	var (
		keyID   = t.SpotifyID
		keyPath = tracksIndex.Tracks[t.SpotifyID]
	)
	if err := os.Rename(keyPath, t.Filename()); err != nil {
		return err
	}

	for _, trackLink := range tracksIndex.Links[keyPath] {
		var (
			trackLinkParts  = strings.Split(trackLink, "/")
			trackLinkFolder = strings.Join(trackLinkParts[:len(trackLinkParts)-1], "/")
			trackLinkName   = trackLinkParts[len(trackLinkParts)-1]
			trackLinkNew    = trackLinkFolder + "/" + t.Filename()
		)
		os.Rename(trackLink, trackLinkNew)
		filepath.Walk(trackLinkFolder, func(path string, info os.FileInfo, err error) error {
			if filepath.Ext(path) != ".m3u" || filepath.Ext(path) != ".pls" {
				return nil
			}

			var (
				playlistLines    = system.FileReadLines(path)
				playlistLinesNew = make([]string, len(playlistLines))
				playlistUpdated  = false
			)
			for _, line := range playlistLines {
				if strings.Contains(line, trackLinkName) {
					line = strings.ReplaceAll(line, trackLinkName, t.Filename())
					playlistUpdated = true
				}
				playlistLinesNew = append(playlistLinesNew, line)
			}

			if playlistUpdated {
				err := system.FileWriteLines(path, playlistLinesNew)
				if err != nil {
					ui.Append(fmt.Sprintf("Unable to update playlist %s: %s", path, err.Error()), cui.ErrorAppend)
				}
			}

			return nil
		})
		tracksIndex.Links[t.Filename()] = append(tracksIndex.Links[t.Filename()], trackLinkNew)
	}
	t.Local = true
	delete(tracksIndex.Links, keyPath)
	tracksIndex.Tracks[keyID] = t.Filename()

	return nil
}
