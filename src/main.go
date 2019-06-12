package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"./cui"
	"./logger"
	"./spotify"
	"./spotitube"
	"./system"
	"./track"
	"./youtube"

	"github.com/0xAX/notificator"
	"github.com/bogem/id3v2"
	"github.com/gosimple/slug"
)

var (
	argFolder                *string
	argPlaylist              *string
	argInvalidateCache       *bool
	argReplaceLocal          *bool
	argFlushMetadata         *bool
	argFlushMissing          *bool
	argFlushDifferent        *bool
	argDisableNormalization  *bool
	argDisablePlaylistFile   *bool
	argPlsFile               *bool
	argDisableLyrics         *bool
	argDisableTimestampFlush *bool
	argDisableUpdateCheck    *bool
	argDisableBrowserOpening *bool
	argDisableIndexing       *bool
	argAuthenticateOutside   *bool
	argInteractive           *bool
	argManualInput           *bool
	argRemoveDuplicates      *bool
	argCleanJunks            *bool
	argLog                   *bool
	argDisableGui            *bool
	argDebug                 *bool
	argSimulate              *bool
	argVersion               *bool
	argFix                   system.PathsArrayFlag

	tracks        track.Tracks
	tracksFailed  track.Tracks
	tracksIndex   = track.TracksIndex{Tracks: make(map[string]string), Links: make(map[string][]string)}
	playlistInfo  *spotify.Playlist
	spotifyClient *spotify.Client
	spotifyUser   string
	spotifyUserID string
	waitGroup     sync.WaitGroup
	waitGroupPool = make(chan bool, spotitube.ConcurrencyLimit)
	waitIndex     = make(chan bool, 1)

	cuiInterface *cui.CUI
	notify       *notificator.Notificator

	procCurrentBin      string
	userLocalConfigPath = spotitube.LocalConfigPath()
	userLocalBin        = fmt.Sprintf("%s/spotitube", userLocalConfigPath)
	userLocalIndex      = fmt.Sprintf("%s/index.gob", userLocalConfigPath)
	userLocalGob        = fmt.Sprintf("%s/%s_%s.gob", userLocalConfigPath, "%s", "%s")
)

func main() {
	procCurrentBin, _ = filepath.Abs(os.Args[0])

	if procCurrentBin != userLocalBin && system.FileExists(userLocalBin) && os.Getenv("FORKED") != "1" {
		syscall.Exec(userLocalBin, os.Args, append(os.Environ(), []string{"FORKED=1"}...))
		mainExit()
	}

	if len(spotify.SpotifyClientID) != 32 && len(os.Getenv("SPOTIFY_ID")) != 32 {
		fmt.Println(fmt.Sprintf("ERROR: Unknown SPOTIFY_ID: please, export SPOTIFY_ID enviroment variable."))
		os.Exit(1)
	}

	if len(spotify.SpotifyClientSecret) != 32 && len(os.Getenv("SPOTIFY_KEY")) != 32 {
		fmt.Println(fmt.Sprintf("ERROR: Unknown SPOTIFY_KEY: please, export SPOTIFY_KEY enviroment variable."))
		os.Exit(1)
	}

	if len(track.GeniusAccessToken) != 64 && len(os.Getenv("GENIUS_TOKEN")) != 64 {
		fmt.Println(fmt.Sprintf("WARNING: Unknown GENIUS_TOKEN: please, export GENIUS_TOKEN enviroment variable, if you wan't to fetch lyrics from Genius provider."))
	}

	argFolder = flag.String("folder", ".", "Folder to sync with music")
	argPlaylist = flag.String("playlist", "none", "Playlist URI to synchronize")
	argInvalidateCache = flag.Bool("invalidate-cache", false, "Manually invalidate library cache, retriggering its fetch from Spotify")
	flag.Var(&argFix, "fix", "Offline song filename(s) which straighten the shot to")
	argReplaceLocal = flag.Bool("replace-local", false, "Replace local library songs if better results get encountered")
	argFlushMetadata = flag.Bool("flush-metadata", false, "Flush metadata informations to already synchronized songs")
	argFlushMissing = flag.Bool("flush-missing", false, "If -flush-metadata toggled, it will just populate empty id3v2 frames, instead of flushing any of those")
	argFlushDifferent = flag.Bool("flush-different", false, "If -flush-metadata toggled, it will just populate id3v2 frames different from the ones calculated by the application, instead of flushing any of those")
	argDisableNormalization = flag.Bool("disable-normalization", false, "Disable songs volume normalization")
	argDisablePlaylistFile = flag.Bool("disable-playlist-file", false, "Disable automatic creation of playlists file")
	argPlsFile = flag.Bool("pls-file", false, "Generate playlist file with .pls instead of .m3u")
	argDisableLyrics = flag.Bool("disable-lyrics", false, "Disable download of songs lyrics and their application into mp3")
	argDisableTimestampFlush = flag.Bool("disable-timestamp-flush", false, "Disable automatic songs files timestamps flush")
	argDisableUpdateCheck = flag.Bool("disable-update-check", false, "Disable automatic update check at startup")
	argDisableBrowserOpening = flag.Bool("disable-browser-opening", false, "Disable automatic browser opening for authentication")
	argDisableIndexing = flag.Bool("disable-indexing", false, "Disable automatic library indexing (used to keep track of tracks names modifications)")
	argAuthenticateOutside = flag.Bool("authenticate-outside", false, "Enable authentication flow to be handled outside this machine")
	argInteractive = flag.Bool("interactive", false, "Enable interactive mode")
	argManualInput = flag.Bool("manual-input", false, "Always manually insert YouTube URL used for songs download")
	argRemoveDuplicates = flag.Bool("remove-duplicates", false, "Remove encountered duplicates from online library/playlist")
	argCleanJunks = flag.Bool("clean-junks", false, "Scan for junks file and clean them")
	argLog = flag.Bool("log", false, "Enable logging into file ./spotitube.log")
	argDebug = flag.Bool("debug", false, "Enable debug messages")
	argSimulate = flag.Bool("simulate", false, "Simulate process flow, without really altering filesystem")
	argVersion = flag.Bool("version", false, "Print version")
	if runtime.GOOS != "windows" {
		argDisableGui = flag.Bool("disable-gui", false, "Disable GUI to reduce noise and increase readability of program flow")
	} else {
		*argDisableGui = true
	}
	flag.Parse()

	if *argVersion {
		fmt.Println(fmt.Sprintf("SpotiTube, version %d.", spotitube.Version))
		os.Exit(0)
	}

	if len(argFix.Paths) > 0 {
		*argReplaceLocal = true
		*argFlushMetadata = true
	}

	if *argManualInput {
		*argInteractive = true
	}

	if !(system.Dir(*argFolder)) {
		fmt.Println(fmt.Sprintf("Chosen music folder does not exist: %s", *argFolder))
		os.Exit(1)
	} else {
		*argFolder, _ = filepath.Abs(*argFolder)
		os.Chdir(*argFolder)
		if user, err := user.Current(); err == nil {
			*argFolder = strings.Replace(*argFolder, user.HomeDir, "~", -1)
		}
	}

	if *argCleanJunks {
		junks := subCleanJunks()
		fmt.Println(fmt.Sprintf("Removed %d junk files.", junks))
		os.Exit(0)
	}

	system.Mkdir(userLocalConfigPath)

	var guiOptions uint64
	if *argDebug {
		guiOptions |= cui.GuiDebugMode
	}
	if *argDisableGui {
		guiOptions |= cui.GuiBareMode
	}
	if *argLog {
		guiOptions |= cui.LogEnable
	}

	var err error
	if cuiInterface, err = cui.Startup(guiOptions); err != nil {
		fmt.Println(fmt.Sprintf("Unable to build user interface: %s", err.Error()))
	}

	cuiInterface.Append(fmt.Sprintf("%s %s", cui.Font("Folder:", cui.StyleBold), *argFolder), cui.PanelLeftTop)
	if *argLog {
		cuiInterface.Append(fmt.Sprintf("%s %s", cui.Font("Log:", cui.StyleBold), logger.LogFilename), cui.PanelLeftTop)
	}
	cuiInterface.Append(fmt.Sprintf("%s %d", cui.Font("Version:", cui.StyleBold), spotitube.Version), cui.PanelLeftBottom)
	if os.Getenv("FORKED") == "1" {
		cuiInterface.Append(fmt.Sprintf("%s %s", cui.Font("Caller:", cui.StyleBold), "automatically updated"), cui.PanelLeftBottom)
	} else {
		cuiInterface.Append(fmt.Sprintf("%s %s", cui.Font("Caller:", cui.StyleBold), "installed"), cui.PanelLeftBottom)
	}
	cuiInterface.Append(fmt.Sprintf("%s %s", cui.Font("Date:", cui.StyleBold), time.Now().Format("2006-01-02 15:04:05")), cui.PanelLeftBottom)
	cuiInterface.Append(fmt.Sprintf("%s %s", cui.Font("URL:", cui.StyleBold), spotitube.VersionRepository), cui.PanelLeftBottom)
	cuiInterface.Append(fmt.Sprintf("%s GPL-3.0", cui.Font("License:", cui.StyleBold)), cui.PanelLeftBottom)

	subCheckDependencies()
	subCheckInternet()
	subCheckUpdate()
	subFetchIndex()

	if !*argDisableIndexing {
		go subAlignIndex()
	}

	go func() {
		<-cuiInterface.CloseChan
		subSafeExit()
	}()
	if *argDisableGui {
		channel := make(chan os.Signal, 1)
		signal.Notify(channel, os.Interrupt)
		go func() {
			for range channel {
				subSafeExit()
			}
		}()
	}

	mainFetch()
}

func mainFetch() {
	if len(argFix.Paths) == 0 {
		var spotifyAuthHost string
		if !*argAuthenticateOutside {
			spotifyAuthHost = "localhost"
		} else {
			*argDisableBrowserOpening = true
			spotifyAuthHost = "spotitube.local"
			cuiInterface.Prompt("Outside authentication enabled: assure \"spotitube.local\" points to this machine.")
		}
		spotifyAuthURL := spotify.BuildAuthURL(spotifyAuthHost)
		cuiInterface.Append(fmt.Sprintf("Authentication URL: %s", spotifyAuthURL.Short), cui.ParagraphAutoReturn)
		if !*argDisableBrowserOpening {
			cuiInterface.Append("Waiting for automatic login process. If wait is too long, manually open that URL.", cui.DebugAppend)
		}
		var err error
		if spotifyClient, err = spotify.Auth(spotifyAuthURL.Full, spotifyAuthHost, !*argDisableBrowserOpening); err != nil {
			cuiInterface.Prompt("Unable to authenticate to spotify.", cui.PromptExit)
		}
		cuiInterface.Append("Authentication completed.")
		spotifyUser, spotifyUserID = spotifyClient.User()
		cuiInterface.Append(fmt.Sprintf("%s %s", cui.Font("Session user:", cui.StyleBold), spotifyUser), cui.PanelLeftTop)

		var (
			tracksOnline          []spotify.Track
			tracksOnlineAlbums    []spotify.Album
			tracksOnlineAlbumsIds []spotify.ID
			tracksErr             error
		)

		if *argPlaylist == "none" {
			userLocalGob = fmt.Sprintf(userLocalGob, spotifyUserID, "library")
			if *argInvalidateCache {
				os.Remove(userLocalGob)
			}
			tracksDump, tracksDumpErr := subFetchGob(userLocalGob)
			if tracksDumpErr != nil {
				cuiInterface.Append(tracksDumpErr.Error(), cui.WarningAppend)
				cuiInterface.Append("Fetching music library...")
				if tracksOnline, tracksErr = spotifyClient.LibraryTracks(); tracksErr != nil {
					cuiInterface.Prompt(fmt.Sprintf("Something went wrong while fetching tracks from library: %s.", tracksErr.Error()), cui.PromptExit)
				}
			} else {
				cuiInterface.Append(fmt.Sprintf("Tracks loaded from cache."))
				cuiInterface.Append(fmt.Sprintf("%s %d/%d (min)", cui.Font("Tracks cache lifetime:", cui.StyleBold), int(time.Since(tracksDump.Time).Minutes()), 30), cui.PanelLeftTop)
				for _, t := range tracksDump.Tracks {
					tracks = append(tracks, t.FlushLocal())
				}
			}
		} else {
			cuiInterface.Append("Fetching playlist data...")
			var playlistErr error
			playlistInfo, playlistErr = spotifyClient.Playlist(*argPlaylist)
			if playlistErr != nil {
				cuiInterface.Prompt("Something went wrong while fetching playlist info.", cui.PromptExit)
			} else {
				cuiInterface.Append(fmt.Sprintf("%s %s", cui.Font("Playlist name:", cui.StyleBold), playlistInfo.Name), cui.PanelLeftTop)
				if len(playlistInfo.Owner.DisplayName) == 0 && len(strings.Split(*argPlaylist, ":")) >= 3 {
					cuiInterface.Append(fmt.Sprintf("%s %s", cui.Font("Playlist owner:", cui.StyleBold), strings.Split(*argPlaylist, ":")[2]), cui.PanelLeftTop)
				} else {
					cuiInterface.Append(fmt.Sprintf("%s %s", cui.Font("Playlist owner:", cui.StyleBold), playlistInfo.Owner.DisplayName), cui.PanelLeftTop)
				}

				userLocalGob = fmt.Sprintf(userLocalGob, playlistInfo.Owner.ID, playlistInfo.Name)
				if *argInvalidateCache {
					os.Remove(userLocalGob)
				}
				tracksDump, tracksDumpErr := subFetchGob(userLocalGob)
				if tracksDumpErr != nil {
					cuiInterface.Append(tracksDumpErr.Error(), cui.WarningAppend)
					cuiInterface.Append(fmt.Sprintf("Getting songs from \"%s\" playlist, by \"%s\"...", playlistInfo.Name, playlistInfo.Owner.DisplayName), cui.StyleBold)
					if tracksOnline, tracksErr = spotifyClient.PlaylistTracks(*argPlaylist); tracksErr != nil {
						cuiInterface.Prompt(fmt.Sprintf("Something went wrong while fetching playlist: %s.", tracksErr.Error()), cui.PromptExit)
					}
				} else {
					cuiInterface.Append(fmt.Sprintf("Tracks loaded from cache."))
					cuiInterface.Append(fmt.Sprintf("%s %d/%d (min)", cui.Font("Tracks cache lifetime:", cui.StyleBold), int(time.Since(tracksDump.Time).Minutes()), 30), cui.PanelLeftTop)
					for _, t := range tracksDump.Tracks {
						tracks = append(tracks, t.FlushLocal())
					}
				}
			}
		}
		for _, t := range tracksOnline {
			tracksOnlineAlbumsIds = append(tracksOnlineAlbumsIds, t.Album.ID)
		}
		if tracksOnlineAlbums, tracksErr = spotifyClient.Albums(tracksOnlineAlbumsIds); tracksErr != nil {
			cuiInterface.Prompt(fmt.Sprintf("Something went wrong while fetching album info: %s.", tracksErr.Error()), cui.PromptExit)
		}

		cuiInterface.Append("Checking which songs need to be downloaded...")
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
				cuiInterface.Append(fmt.Sprintf("Ignored song duplicate \"%s\" by \"%s\".", tracksOnline[trackIndex].SimpleTrack.Name, tracksOnline[trackIndex].SimpleTrack.Artists[0].Name), cui.WarningAppend)
				tracksDuplicates = append(tracksDuplicates, trackID)
			}
		}

		if *argRemoveDuplicates && len(tracksDuplicates) > 0 {
			if *argPlaylist == "none" {
				if removeErr := spotifyClient.RemoveLibraryTracks(tracksDuplicates); removeErr != nil {
					cuiInterface.Prompt(fmt.Sprintf("Something went wrong while removing %d duplicates: %s.", len(tracksDuplicates), removeErr.Error()))
				} else {
					cuiInterface.Append(fmt.Sprintf("%d duplicate tracks correctly removed from library.", len(tracksDuplicates)))
				}
			} else {
				if removeErr := spotifyClient.RemovePlaylistTracks(*argPlaylist, tracksDuplicates); removeErr != nil {
					cuiInterface.Prompt(fmt.Sprintf("Something went wrong while removing %d duplicates: %s.", len(tracksDuplicates), removeErr.Error()))
				} else {
					cuiInterface.Append(fmt.Sprintf("%d duplicate tracks correctly removed from playlist.", len(tracksDuplicates)))
				}
			}
		}

		if dumpErr := system.DumpGob(userLocalGob, track.TracksDump{Tracks: tracks, Time: time.Now()}); dumpErr != nil {
			cuiInterface.Append(fmt.Sprintf("Unable to cache tracks: %s", dumpErr.Error()), cui.WarningAppend)
		}

		cuiInterface.Append(fmt.Sprintf("%s %d", cui.Font("Songs duplicates:", cui.StyleBold), len(tracksDuplicates)), cui.PanelLeftTop)
		cuiInterface.Append(fmt.Sprintf("%s %d", cui.Font("Songs online:", cui.StyleBold), len(tracks)), cui.PanelLeftTop)
		cuiInterface.Append(fmt.Sprintf("%s %d", cui.Font("Songs offline:", cui.StyleBold), tracks.CountOffline()), cui.PanelLeftTop)
		cuiInterface.Append(fmt.Sprintf("%s %d", cui.Font("Songs missing:", cui.StyleBold), tracks.CountOnline()), cui.PanelLeftTop)

		<-waitIndex
		close(waitIndex)
	} else {
		cuiInterface.Append(fmt.Sprintf("%s %d", cui.Font("Fix song(s):", cui.StyleBold), len(argFix.Paths)), cui.PanelLeftTop)
		for _, fixTrack := range argFix.Paths {
			if t, trackErr := track.OpenLocalTrack(fixTrack); trackErr != nil {
				cuiInterface.Prompt(fmt.Sprintf("Something went wrong: %s.", trackErr.Error()), cui.PromptExit)
				mainExit()
			} else {
				cuiInterface.Append(fmt.Sprintf("%+v\n", t), cui.DebugAppend)
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
		cuiInterface.Prompt("No song needs to be downloaded.", cui.PromptExit)
		mainExit()
	}
}

func mainSearch() {
	defer mainExit()
	defer subCleanJunks()

	cuiInterface.ProgressMax = len(tracks)

	songsFetch, songsFlush, songsIgnore := subCountSongs()
	cuiInterface.Append(fmt.Sprintf("%d will be downloaded, %d flushed and %d ignored", songsFetch, songsFlush, songsIgnore))

	for trackIndex, t := range tracks {
		cuiInterface.ProgressHalfIncrease()
		cuiInterface.Append(fmt.Sprintf("%d/%d: \"%s\"", trackIndex+1, len(tracks), t.Filename()), cui.StyleBold)

		if trackPath, ok := tracksIndex.Tracks[t.SpotifyID]; ok {
			if trackPath != t.Filename() {
				cuiInterface.Append(fmt.Sprintf("Track %s has been renamed: moving local one to %s", t.SpotifyID, t.Filename()))
				if err := subTrackRename(&t); err != nil {
					cuiInterface.Append(fmt.Sprintf("Unable to rename: %s", err.Error()), cui.ErrorAppend)
				}
			}
		}

		if subIfSongSearch(t) {
			var (
				youTubeTrack         = youtube.Track{Track: &t}
				youTubeTracks        = youtube.Tracks{}
				youTubeTracksErr     error
				youTubeTrackPickAuto bool
				youTubeTrackPick     bool
			)
			if !*argManualInput {
				youTubeTracks, youTubeTracksErr = youtube.QueryTracks(&t)
				if youTubeTracksErr != nil {
					cuiInterface.Append(fmt.Sprintf("Something went wrong while searching for \"%s\" track: %s.", t.Basename(), youTubeTracksErr.Error()), cui.WarningAppend)
					tracksFailed = append(tracksFailed, t)
					cuiInterface.ProgressHalfIncrease()
					continue
				}
				for _, youTubeTrackLoopEl := range youTubeTracks {
					cuiInterface.Append(fmt.Sprintf("Result met: ID: %s,\nTitle: %s,\nUser: %s,\nDuration: %d.",
						youTubeTrackLoopEl.ID, youTubeTrackLoopEl.Title, youTubeTrackLoopEl.User, youTubeTrackLoopEl.Duration), cui.DebugAppend)

					youTubeTrackPickAuto, youTubeTrackPick = subMatchResult(t, youTubeTrackLoopEl)
					if subIfPickFromAns(youTubeTrackPickAuto, youTubeTrackPick) {
						cuiInterface.Append(fmt.Sprintf("Video \"%s\" is good to go for \"%s\".", youTubeTrackLoopEl.Title, t.Basename()))
						youTubeTrack = youTubeTrackLoopEl
						break
					}
				}
			}

			subCondManualInputURL(&youTubeTrack)
			if youTubeTrack.URL == "" {
				youTubeTrack = youtube.Track{}
			} else {
				youTubeTrack.Track = &t
			}

			if youTubeTrack == (youtube.Track{}) {
				cuiInterface.Append(fmt.Sprintf("Video for \"%s\" not found.", t.Basename()), cui.ErrorAppend)
				tracksFailed = append(tracksFailed, t)
				cuiInterface.ProgressHalfIncrease()
				continue
			}

			if *argSimulate {
				cuiInterface.Append(fmt.Sprintf("I would like to download \"%s\" for \"%s\" track, but I'm just simulating.", youTubeTrack.URL, t.Basename()))
				cuiInterface.ProgressHalfIncrease()
				continue
			} else if *argReplaceLocal {
				if t.URL == youTubeTrack.URL && !youTubeTrackPick {
					cuiInterface.Append(fmt.Sprintf("Track \"%s\" is still the best result I can find.", t.Basename()))
					cuiInterface.Append(fmt.Sprintf("Local track origin URL %s is the same as YouTube chosen one %s.", t.URL, youTubeTrack.URL), cui.DebugAppend)
					cuiInterface.ProgressHalfIncrease()
					continue
				} else {
					t.URL = ""
					t.Local = false
				}
			}

			cuiInterface.Append(fmt.Sprintf("Going to download \"%s\" from %s...", youTubeTrack.Title, youTubeTrack.URL))
			err := youTubeTrack.Download()
			if err != nil {
				cuiInterface.Append(fmt.Sprintf("Something went wrong downloading \"%s\": %s.", t.Basename(), err.Error()), cui.WarningAppend)
				tracksFailed = append(tracksFailed, t)
				cuiInterface.ProgressHalfIncrease()
				continue
			} else {
				t.URL = youTubeTrack.URL
			}
		}

		if !subIfSongProcess(t) {
			cuiInterface.ProgressHalfIncrease()
			continue
		}

		subCondSequentialDo(&t)

		cuiInterface.Append(fmt.Sprintf("Launching song processing jobs..."))
		waitGroup.Add(1)
		go subParallelSongProcess(t, &waitGroup)
		if *argDebug {
			waitGroup.Wait()
		}
	}
	waitGroup.Wait()

	subCondPlaylistFileWrite()
	subCondTimestampFlush()
	subWriteIndex()

	junks := subCleanJunks()
	cuiInterface.Append(fmt.Sprintf("Removed %d junk files.", junks))

	close(waitGroupPool)
	waitGroup.Wait()
	cuiInterface.ProgressFill()

	cuiInterface.Append(fmt.Sprintf("%d tracks failed to synchronize.", len(tracksFailed)))
	for _, t := range tracksFailed {
		cuiInterface.Append(fmt.Sprintf(" - \"%s\"", t.Basename()))
	}

	var (
		notify = notificator.New(notificator.Options{
			DefaultIcon: "emblem-downloads",
			AppName:     "SpotiTube",
		})
		notifyTitle   string
		notifyContent string
	)
	if *argPlaylist == "none" {
		notifyTitle = "Library synchronization"
	} else {
		notifyTitle = fmt.Sprintf("%s playlist synchronization", playlistInfo.Name)
	}
	if len(tracksFailed) > 0 {
		notifyContent = fmt.Sprintf("%d track(s) synced, %d failed.", len(tracks)-len(tracksFailed), len(tracksFailed))
	} else {
		notifyContent = fmt.Sprintf("%d track(s) correctly synced.", len(tracks))
	}
	notify.Push(notifyTitle, notifyContent, "", notificator.UR_NORMAL)

	cuiInterface.Prompt("Synchronization completed.", cui.PromptExit)
}

func mainExit(delay ...time.Duration) {
	if len(delay) > 0 {
		time.Sleep(delay[0])
	}

	os.Exit(0)
}

func subCheckDependencies() {
	for _, commandName := range []string{"youtube-dl", "ffmpeg"} {
		_, err := exec.LookPath(commandName)
		if err != nil {
			cuiInterface.Prompt(fmt.Sprintf("Are you sure %s is asctually installed?", commandName), cui.PromptExit)
		} else {
			var (
				commandOut          bytes.Buffer
				commandVersionValue = "?"
				commandVersionRegex = "\\d+\\.\\d+\\.\\d+"
			)
			if versionRegex, versionRegexErr := regexp.Compile(commandVersionRegex); versionRegexErr != nil {
				commandVersionValue = "Regex compile failure"
			} else {
				commandObj := exec.Command(commandName, []string{"--version"}...)
				commandObj.Stdout = &commandOut
				commandObj.Stderr = &commandOut
				_ = commandObj.Run()
				if commandVersionRegValue := versionRegex.FindString(commandOut.String()); len(commandVersionRegValue) > 0 {
					commandVersionValue = commandVersionRegValue
				}
			}
			cuiInterface.Append(fmt.Sprintf("%s %s", cui.Font(fmt.Sprintf("Version %s:", commandName), cui.StyleBold), commandVersionValue), cui.PanelLeftTop)
		}
	}
}

func subCheckInternet() {
	client := http.Client{
		Timeout: time.Second * spotitube.HTTPTimeout,
	}
	req, _ := http.NewRequest("GET", "http://clients3.google.com/generate_204", nil)
	_, err := client.Do(req)
	if err != nil {
		cuiInterface.Prompt("Are you sure you're connected to the internet?", cui.PromptExit)
	}
}

func subCheckUpdate() {
	if !*argDisableUpdateCheck {
		type OnlineVersion struct {
			Name string `json:"name"`
		}
		versionClient := http.Client{
			Timeout: time.Second * spotitube.HTTPTimeout,
		}
		versionRequest, versionError := http.NewRequest(http.MethodGet, spotitube.VersionOrigin, nil)
		if versionError != nil {
			cuiInterface.Append(fmt.Sprintf("Unable to compile version request: %s", versionError.Error()), cui.WarningAppend)
		} else {
			versionResponse, versionError := versionClient.Do(versionRequest)
			if versionError != nil {
				cuiInterface.Append(fmt.Sprintf("Unable to read response from version request: %s", versionError.Error()), cui.WarningAppend)
			} else {
				versionResponseBody, versionError := ioutil.ReadAll(versionResponse.Body)
				if versionError != nil {
					cuiInterface.Append(fmt.Sprintf("Unable to get response body: %s", versionError.Error()), cui.WarningAppend)
				} else {
					versionData := OnlineVersion{}
					versionError = json.Unmarshal(versionResponseBody, &versionData)
					if versionError != nil {
						cuiInterface.Append(fmt.Sprintf("Unable to parse json from response body: %s", versionError.Error()), cui.WarningAppend)
					} else {
						versionValue := 0
						versionRegex, versionError := regexp.Compile("[^0-9]+")
						if versionError != nil {
							cuiInterface.Append(fmt.Sprintf("Unable to compile regex needed to parse version: %s", versionError.Error()), cui.WarningAppend)
						} else {
							versionValue, versionError = strconv.Atoi(versionRegex.ReplaceAllString(versionData.Name, ""))
							if versionError != nil {
								cuiInterface.Append(fmt.Sprintf("Unable to fetch latest version value: %s", versionError.Error()), cui.WarningAppend)
							} else if versionValue != spotitube.Version {
								cuiInterface.Append(fmt.Sprintf("Going to update from %d to %d version.", spotitube.Version, versionValue))
								subUpdateSoftware(versionResponseBody)
							}
							cuiInterface.Append(fmt.Sprintf("Actual version %d, online version %d.", spotitube.Version, versionValue), cui.DebugAppend)
						}
					}
				}
			}
		}
	}
}

func subFetchIndex() {
	if !system.FileExists(userLocalIndex) {
		cuiInterface.Append("No track index has been found.", cui.DebugAppend)
		return
	}
	cuiInterface.Append("Fetching local index...", cui.DebugAppend)
	system.FetchGob(userLocalIndex, tracksIndex)
}

func subWriteIndex() {
	cuiInterface.Append(fmt.Sprintf("Writing %d entries index...", len(tracksIndex.Tracks)), cui.DebugAppend)
	if writeErr := system.DumpGob(userLocalIndex, tracksIndex); writeErr != nil {
		cuiInterface.Append(fmt.Sprintf("Unable to write tracks index: %s", writeErr.Error()), cui.WarningAppend)
	}
}

func subAlignIndex() {
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		if linkPath, err := os.Readlink(path); err == nil {
			cuiInterface.Append(fmt.Sprintf("Index link: path %s", path), cui.DebugAppend)
			tracksIndex.Links[linkPath] = append(tracksIndex.Links[linkPath], path)
		} else if filepath.Ext(path) == ".mp3" {
			cuiInterface.Append(fmt.Sprintf("Index: path %s", path), cui.DebugAppend)
			spotifyID := track.GetTag(path, track.ID3FrameSpotifyID)
			if len(spotifyID) > 0 {
				tracksIndex.Tracks[spotifyID] = path
			} else {
				cuiInterface.Append(fmt.Sprintf("Index: no ID found. Ignoring %s...", path), cui.DebugAppend)
			}
		}

		return nil
	})
	waitIndex <- true
	cuiInterface.Append(fmt.Sprintf("Indexing finished: %d tracks indexed and %d linked.", len(tracksIndex.Tracks), len(tracksIndex.Links)))
}

func subUpdateSoftware(latestRelease []byte) {
	var (
		api             map[string]interface{}
		binaryName      string
		binaryURL       string
		binaryTempFname string
	)
	unmarshalErr := json.Unmarshal([]byte(latestRelease), &api)
	if unmarshalErr != nil {
		cuiInterface.Append(fmt.Sprintf("Unable to unmarshal Github latest relase data: %s", unmarshalErr.Error()), cui.WarningAppend)
		return
	}
	for _, asset := range api["assets"].([]interface{}) {
		binaryName = asset.(map[string]interface{})["name"].(string)
		if filepath.Ext(binaryName) == ".bin" {
			binaryURL = asset.(map[string]interface{})["browser_download_url"].(string)
			break
		}
	}

	binaryTempFname = fmt.Sprintf("/tmp/.%s", binaryName)
	binaryOutput, err := os.Create(binaryTempFname)
	if err != nil {
		cuiInterface.Append(fmt.Sprintf("Unable to create temporary updated binary file: %s", err.Error()), cui.WarningAppend)
		return
	}
	defer binaryOutput.Close()

	cuiInterface.Append(fmt.Sprintf("Downloading update from %s...", binaryURL))
	binaryPayload, err := http.Get(binaryURL)
	if err != nil {
		cuiInterface.Append(fmt.Sprintf("Unable to download from %s: %s", binaryURL, err.Error()), cui.WarningAppend)
		return
	}
	defer binaryPayload.Body.Close()

	_, err = io.Copy(binaryOutput, binaryPayload.Body)
	if err != nil {
		cuiInterface.Append(fmt.Sprintf("Error while downloading from %s: %s", binaryURL, err.Error()), cui.WarningAppend)
		return
	}

	if user, err := user.Current(); err == nil {
		var (
			binaryFolder = fmt.Sprintf("%s/.spotitube", user.HomeDir)
			binaryFname  = fmt.Sprintf("%s/spotitube", binaryFolder)
		)
		err = system.Mkdir(binaryFolder)
		if err != nil {
			cuiInterface.Append(fmt.Sprintf("Unable to create binary container folder at %s: %s", binaryFolder, err.Error()), cui.WarningAppend)
			return
		}
		os.Remove(binaryFname)
		err = system.FileCopy(binaryTempFname, binaryFname)
		if err != nil {
			cuiInterface.Append(fmt.Sprintf("Unable to persist new binary to %s: %s", binaryFname, err.Error()), cui.WarningAppend)
			return
		}
		os.Remove(binaryTempFname)

		err = os.Chmod(binaryFname, 0755)
		if err != nil {
			cuiInterface.Append(fmt.Sprintf("Unable to make %s executable: %s", binaryFname, err.Error()), cui.WarningAppend)
			return
		}

		err = syscall.Exec(binaryFname, os.Args, os.Environ())
		if err != nil {
			cuiInterface.Append(fmt.Sprintf("Unable to exec updated instance: %s", err.Error()), cui.ErrorAppend)
		}
		mainExit()
	} else {
		return
	}
}

func subParallelSongProcess(t track.Track, wg *sync.WaitGroup) {
	defer cuiInterface.ProgressHalfIncrease()
	defer wg.Done()
	<-waitGroupPool

	if !t.Local && !*argDisableNormalization {
		subSongNormalize(t)
	}

	if !system.FileExists(t.FilenameTemporary()) && system.FileExists(t.Filename()) {
		if err := system.FileCopy(t.Filename(), t.FilenameTemporary()); err != nil {
			cuiInterface.Append(fmt.Sprintf("Unable to prepare song for getting its metadata flushed: %s", err.Error()), cui.WarningAppend)
			return
		}
	}

	if (t.Local && *argFlushMetadata) || !t.Local {
		subSongFlushMetadata(t)
	}

	os.Remove(t.Filename())
	err := os.Rename(t.FilenameTemporary(), t.Filename())
	if err != nil {
		cuiInterface.Append(fmt.Sprintf("Unable to move song to its final path: %s", err.Error()), cui.WarningAppend)
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
	cuiInterface.Append(fmt.Sprintf("Getting max_volume value: \"%s %s\"...", commandCmd, strings.Join(commandArgs, " ")), cui.DebugAppend)
	commandObj := exec.Command(commandCmd, commandArgs...)
	commandObj.Stderr = &commandOut
	commandErr = commandObj.Run()
	if commandErr != nil {
		cuiInterface.Append(fmt.Sprintf("Unable to use ffmpeg to pull max_volume song value: %s.", commandOut.String()), cui.WarningAppend)
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
		cuiInterface.Append(fmt.Sprintf("Unable to pull max_volume delta to be applied along with song volume normalization: %s.", normalizationDelta), cui.WarningAppend)
		normalizationDelta = "0.0"
	}
	commandArgs = []string{"-i", t.FilenameTemporary(), "-af", "volume=+" + normalizationDelta + "dB", "-b:a", "320k", "-y", normalizationFile}
	cuiInterface.Append(fmt.Sprintf("Compensating volume by %sdB...", normalizationDelta), cui.DebugAppend)
	cuiInterface.Append(fmt.Sprintf("Increasing audio quality for: %s...", t.Basename()), cui.DebugAppend)
	cuiInterface.Append(fmt.Sprintf("Firing command: \"%s %s\"...", commandCmd, strings.Join(commandArgs, " ")), cui.DebugAppend)
	if _, commandErr = exec.Command(commandCmd, commandArgs...).Output(); commandErr != nil {
		cuiInterface.Append(fmt.Sprintf("Something went wrong while normalizing song \"%s\" volume: %s", t.Basename(), commandErr.Error()), cui.WarningAppend)
	}
	os.Remove(t.FilenameTemporary())
	os.Rename(normalizationFile, t.FilenameTemporary())
}

func subSongFlushMetadata(t track.Track) {
	trackMp3, err := id3v2.Open(t.FilenameTemporary(), id3v2.Options{Parse: true})
	if err != nil {
		cuiInterface.Append(fmt.Sprintf("Something bad happened while opening: %s", err.Error()), cui.WarningAppend)
	} else {
		cuiInterface.Append(fmt.Sprintf("Fixing metadata for \"%s\"...", t.Basename()), cui.DebugAppend)
		if !*argFlushMissing && !*argFlushDifferent {
			trackMp3.DeleteAllFrames()
		}
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
		subCondFlushID3FrameYouTubeURL(t, trackMp3)
		subCondFlushID3FrameDuration(t, trackMp3)
		subCondFlushID3FrameSpotifyID(t, trackMp3)
		subCondFlushID3FrameLyrics(t, trackMp3)
		trackMp3.Save()
	}
	trackMp3.Close()
}

func subCondFlushID3FrameTitle(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Title) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameTitle))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameTitle) != t.Title)) {
		cuiInterface.Append("Inflating title metadata...", cui.DebugAppend)
		trackMp3.SetTitle(t.Title)
	}
}

func subCondFlushID3FrameSong(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Song) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameSong))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameSong) != t.Song)) {
		cuiInterface.Append("Inflating song metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "song",
			Text:        t.Song,
		})
	}
}

func subCondFlushID3FrameArtist(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Artist) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameArtist))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameArtist) != t.Artist)) {
		cuiInterface.Append("Inflating artist metadata...", cui.DebugAppend)
		trackMp3.SetArtist(t.Artist)
	}
}

func subCondFlushID3FrameAlbum(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Album) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameAlbum))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameAlbum) != t.Album)) {
		cuiInterface.Append("Inflating album metadata...", cui.DebugAppend)
		trackMp3.SetAlbum(t.Album)
	}
}

func subCondFlushID3FrameGenre(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Genre) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameGenre))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameGenre) != t.Genre)) {
		cuiInterface.Append("Inflating genre metadata...", cui.DebugAppend)
		trackMp3.SetGenre(t.Genre)
	}
}

func subCondFlushID3FrameYear(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Year) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameYear))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameYear) != t.Year)) {
		cuiInterface.Append("Inflating year metadata...", cui.DebugAppend)
		trackMp3.SetYear(t.Year)
	}
}

func subCondFlushID3FrameFeaturings(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Featurings) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameFeaturings))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameFeaturings) != strings.Join(t.Featurings, "|"))) {
		cuiInterface.Append("Inflating featurings metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "featurings",
			Text:        strings.Join(t.Featurings, "|"),
		})
	}
}

func subCondFlushID3FrameTrackNumber(t track.Track, trackMp3 *id3v2.Tag) {
	if t.TrackNumber > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameTrackNumber))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameTrackNumber) != fmt.Sprintf("%d", t.TrackNumber))) {
		cuiInterface.Append("Inflating track number metadata...", cui.DebugAppend)
		trackMp3.AddFrame(trackMp3.CommonID("Track number/Position in set"),
			id3v2.TextFrame{
				Encoding: id3v2.EncodingUTF8,
				Text:     strconv.Itoa(t.TrackNumber),
			})
	}
}

func subCondFlushID3FrameTrackTotals(t track.Track, trackMp3 *id3v2.Tag) {
	if t.TrackTotals > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameTrackTotals))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameTrackTotals) != fmt.Sprintf("%d", t.TrackTotals))) {
		cuiInterface.Append("Inflating total tracks number metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "trackTotals",
			Text:        fmt.Sprintf("%d", t.TrackTotals),
		})
	}
}

func subCondFlushID3FrameArtwork(t track.Track, trackMp3 *id3v2.Tag) {
	if system.FileExists(t.FilenameArtwork()) &&
		(!*argFlushMissing || (*argFlushMissing && !t.HasID3Frame(track.ID3FrameArtwork))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameArtworkURL) != t.Image)) {
		trackArtworkReader, trackArtworkErr := ioutil.ReadFile(t.FilenameArtwork())
		if trackArtworkErr != nil {
			cuiInterface.Append(fmt.Sprintf("Unable to read artwork file: %s", trackArtworkErr.Error()), cui.WarningAppend)
		} else {
			cuiInterface.Append("Inflating artwork metadata...", cui.DebugAppend)
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
	if len(t.Image) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameArtworkURL))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameArtworkURL) != t.Image)) {
		cuiInterface.Append("Inflating artwork url metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "artwork",
			Text:        t.Image,
		})
	}
}

func subCondFlushID3FrameYouTubeURL(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.URL) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameYouTubeURL))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameYouTubeURL) != t.URL)) {
		cuiInterface.Append("Inflating youtube origin url metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "youtube",
			Text:        t.URL,
		})
	}
}

func subCondFlushID3FrameDuration(t track.Track, trackMp3 *id3v2.Tag) {
	if t.Duration > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameDuration))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameDuration) != fmt.Sprintf("%d", t.Duration))) {
		cuiInterface.Append("Inflating duration metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "duration",
			Text:        fmt.Sprintf("%d", t.Duration),
		})
	}
}

func subCondFlushID3FrameSpotifyID(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.SpotifyID) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameSpotifyID))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameSpotifyID) != t.SpotifyID)) {
		cuiInterface.Append("Inflating Spotify ID metadata...", cui.DebugAppend)
		trackMp3.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "spotifyid",
			Text:        t.SpotifyID,
		})
	}
}

func subCondFlushID3FrameLyrics(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Lyrics) > 0 && !*argDisableLyrics &&
		(!*argFlushMissing || (*argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameLyrics))) &&
		(!*argFlushDifferent || (*argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameLyrics) != t.Lyrics)) {
		cuiInterface.Append("Inflating lyrics metadata...", cui.DebugAppend)
		trackMp3.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
			Encoding:          id3v2.EncodingUTF8,
			Language:          "eng",
			ContentDescriptor: t.Title,
			Lyrics:            t.Lyrics,
		})
	}
}

func subIfSongSearch(t track.Track) bool {
	return !t.Local || *argReplaceLocal || *argSimulate
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
	if *argReplaceLocal {
		songsFetch = len(tracks)
		songsFlush = songsFetch
	} else if *argFlushMetadata {
		songsFetch = tracks.CountOnline()
		songsFlush = len(tracks)
	} else {
		songsFetch = tracks.CountOnline()
		songsFlush = songsFetch
		songsIgnore = tracks.CountOffline()
	}
	return songsFetch, songsFlush, songsIgnore
}

func subMatchResult(t track.Track, youTubeTrack youtube.Track) (bool, bool) {
	var (
		ansInput     bool
		ansAutomated bool
		ansErr       error
	)
	ansErr = youTubeTrack.Match(t)
	ansAutomated = bool(ansErr == nil)
	if *argInteractive {
		ansInput = cuiInterface.Prompt(fmt.Sprintf("Do you want to download the following video for \"%s\"?\n"+
			"ID: %s\nTitle: %s\nUser: %s\nDuration: %d\nURL: %s\nResult is matching: %s",
			t.Basename(), youTubeTrack.ID, youTubeTrack.Title, youTubeTrack.User,
			youTubeTrack.Duration, youTubeTrack.URL, strconv.FormatBool(ansAutomated)), cui.PromptBinary)
	}
	return ansAutomated, ansInput
}

func subIfPickFromAns(ansAutomated bool, ansInput bool) bool {
	return (!*argInteractive && ansAutomated) || (*argInteractive && ansInput)
}

func subCondManualInputURL(youTubeTrack *youtube.Track) {
	if *argInteractive && youTubeTrack.URL == "" {
		inputURL := cuiInterface.PromptInputMessage(fmt.Sprintf("Please, manually enter URL for \"%s\"", youTubeTrack.Track.Basename()), cui.PromptInput)
		if len(inputURL) > 0 {
			if err := youtube.ValidateURL(inputURL); err == nil {
				youTubeTrack.Title = "input video"
				youTubeTrack.URL = inputURL
			} else {
				cuiInterface.Prompt(fmt.Sprintf("Something went wrong: %s", err.Error()))
			}
		}
	}
}

func subIfSongProcess(t track.Track) bool {
	return !t.Local || *argFlushMetadata || *argReplaceLocal
}

func subCondSequentialDo(t *track.Track) {
	if (t.Local && *argFlushMetadata) || !t.Local {
		subCondLyricsFetch(t)
		subCondArtworkDownload(t)
	}
}

func subCondLyricsFetch(t *track.Track) {
	if !*argDisableLyrics &&
		(!*argFlushMissing || (*argFlushMissing && !t.HasID3Frame(track.ID3FrameLyrics))) {
		cuiInterface.Append(fmt.Sprintf("Fetching song \"%s\" lyrics...", t.Basename()), cui.DebugAppend)
		lyricsErr := t.SearchLyrics()
		if lyricsErr != nil {
			cuiInterface.Append(fmt.Sprintf("Something went wrong while searching for song lyrics: %s", lyricsErr.Error()), cui.WarningAppend)
		} else {
			cuiInterface.Append(fmt.Sprintf("Song lyrics found."), cui.DebugAppend)
		}
	}
}

func subCondArtworkDownload(t *track.Track) {
	if len(t.Image) > 0 && !system.FileExists(t.FilenameArtwork()) &&
		(!*argFlushMissing || (*argFlushMissing && !t.HasID3Frame(track.ID3FrameArtwork))) {
		cuiInterface.Append(fmt.Sprintf("Downloading song \"%s\" artwork at %s...", t.Basename(), t.Image), cui.DebugAppend)
		var commandOut bytes.Buffer
		commandCmd := "ffmpeg"
		commandArgs := []string{"-i", t.Image, "-q:v", "1", "-n", t.FilenameArtwork()}
		commandObj := exec.Command(commandCmd, commandArgs...)
		commandObj.Stderr = &commandOut
		if err := commandObj.Run(); err != nil {
			cuiInterface.Append(fmt.Sprintf("Unable to download artwork file \"%s\": %s", t.Image, commandOut.String()), cui.WarningAppend)
		}
	}
}

func subCondTimestampFlush() {
	if !*argDisableTimestampFlush {
		cuiInterface.Append("Flushing files timestamps...")
		now := time.Now().Local().Add(time.Duration(-1*len(tracks)) * time.Minute)
		for _, t := range tracks {
			if !system.FileExists(t.Filename()) {
				continue
			}
			if err := os.Chtimes(t.Filename(), now, now); err != nil {
				cuiInterface.Append(fmt.Sprintf("Unable to flush timestamp on %s", t.Filename()), cui.WarningAppend)
			}
			now = now.Add(1 * time.Minute)
		}
	}
}

func subCondPlaylistFileWrite() {
	if !*argSimulate && !*argDisablePlaylistFile && *argPlaylist != "none" {
		var (
			playlistFolder  = slug.Make(playlistInfo.Name)
			playlistFname   = fmt.Sprintf("%s/%s", playlistFolder, playlistInfo.Name)
			playlistContent string
		)

		if !*argPlsFile {
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
					cuiInterface.Append(fmt.Sprintf("Unable to create symlink for \"%s\" in %s: %s", t.Filename(), playlistFolder, err.Error()), cui.ErrorAppend)
				}
			}
		}
		os.Chdir("..")

		cuiInterface.Append(fmt.Sprintf("Creating playlist file at %s...", playlistFname))
		if system.FileExists(playlistFname) {
			os.Remove(playlistFname)
		}

		if !*argPlsFile {
			playlistContent = "#EXTM3U\n"
			for trackIndex := len(tracks) - 1; trackIndex >= 0; trackIndex-- {
				t := tracks[trackIndex]
				if system.FileExists(t.Filename()) {
					playlistContent += "#EXTINF:" + strconv.Itoa(t.Duration) + "," + t.Filename() + "\n" +
						"./" + t.Filename() + "\n"
				}
			}
		} else {
			cuiInterface.Append("Creating playlist PLS file...")
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
			cuiInterface.Append(fmt.Sprintf("Unable to create M3U file: %s", playlistErr.Error()), cui.WarningAppend)
		} else {
			defer playlistFile.Close()
			_, playlistErr := playlistFile.WriteString(playlistContent)
			playlistFile.Sync()
			if playlistErr != nil {
				cuiInterface.Append(fmt.Sprintf("Unable to write M3U file: %s", playlistErr.Error()), cui.WarningAppend)
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
					cuiInterface.Append(fmt.Sprintf("Unable to update playlist %s: %s", path, err.Error()), cui.ErrorAppend)
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

func subCleanJunks() int {
	var removedJunks int
	for _, junkType := range track.JunkWildcards() {
		junkPaths, err := filepath.Glob(junkType)
		if err != nil {
			continue
		}
		for _, junkPath := range junkPaths {
			os.Remove(junkPath)
			removedJunks++
		}
	}
	return removedJunks
}

func subSafeExit() {
	fmt.Println("Signal captured: cleaning up temporary files...")
	junks := subCleanJunks()
	fmt.Println(fmt.Sprintf("Cleaned up %d files. Exiting.", junks))
	mainExit()
}
