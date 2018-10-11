package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	spttb_gui "gui"
	spttb_logger "logger"
	spttb_spotify "spotify"
	spttb_system "system"
	spttb_track "track"
	spttb_youtube "youtube"

	"github.com/0xAX/notificator"
	id3 "github.com/bogem/id3v2"
	"github.com/kennygrant/sanitize"
	api "github.com/zmb3/spotify"
)

var (
	argFolder                *string
	argPlaylist              *string
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
	argInteractive           *bool
	argManualInput           *bool
	argCleanJunks            *bool
	argLog                   *bool
	argDisableGui            *bool
	argDebug                 *bool
	argSimulate              *bool
	argVersion               *bool
	argFix                   spttb_system.PathsArrayFlag

	tracks        spttb_track.Tracks
	tracksFailed  spttb_track.Tracks
	playlistInfo  *api.FullPlaylist
	spotifyClient *spttb_spotify.Spotify = spttb_spotify.NewClient()
	waitGroup     sync.WaitGroup
	waitGroupPool = make(chan bool, spttb_system.ConcurrencyLimit)

	gui    *spttb_gui.Gui
	notify *notificator.Notificator
)

func main() {
	var (
		currentUser, _ = user.Current()
		currentBin, _  = filepath.Abs(os.Args[0])
		userLocalBin   = fmt.Sprintf("%s/.spotitube/spotitube", currentUser.HomeDir)
	)
	if currentBin != userLocalBin && spttb_system.FileExists(userLocalBin) && os.Getenv("FORKED") != "1" {
		syscall.Exec(userLocalBin, os.Args, append(os.Environ(), []string{"FORKED=1"}...))
		mainExit()
	}

	argFolder = flag.String("folder", ".", "Folder to sync with music")
	argPlaylist = flag.String("playlist", "none", "Playlist URI to synchronize")
	flag.Var(&argFix, "fix", "Offline song filename(s) which straighten the shot to")
	argReplaceLocal = flag.Bool("replace-local", false, "Replace local library songs if better results get encountered")
	argFlushMetadata = flag.Bool("flush-metadata", false, "Flush metadata informations to already synchronized songs")
	argFlushMissing = flag.Bool("flush-missing", false, "If -flush-metadata toggled, it will just populate empty id3 frames, instead of flushing any of those")
	argFlushDifferent = flag.Bool("flush-different", false, "If -flush-metadata toggled, it will just populate id3 frames different from the ones calculated by the application, instead of flushing any of those")
	argDisableNormalization = flag.Bool("disable-normalization", false, "Disable songs volume normalization")
	argDisablePlaylistFile = flag.Bool("disable-playlist-file", false, "Disable automatic creation of playlists file")
	argPlsFile = flag.Bool("pls-file", false, "Generate playlist file with .pls instead of .m3u")
	argDisableLyrics = flag.Bool("disable-lyrics", false, "Disable download of songs lyrics and their application into mp3")
	argDisableTimestampFlush = flag.Bool("disable-timestamp-flush", false, "Disable automatic songs files timestamps flush")
	argDisableUpdateCheck = flag.Bool("disable-update-check", false, "Disable automatic update check at startup")
	argInteractive = flag.Bool("interactive", false, "Enable interactive mode")
	argManualInput = flag.Bool("manual-input", false, "Always manually insert YouTube URL used for songs download")
	argCleanJunks = flag.Bool("clean-junks", false, "Scan for junks file and clean them")
	argLog = flag.Bool("log", false, "Enable logging into file ./spotitube.log")
	argDisableGui = flag.Bool("disable-gui", false, "Disable GUI to reduce noise and increase readability of program flow")
	argDebug = flag.Bool("debug", false, "Enable debug messages")
	argSimulate = flag.Bool("simulate", false, "Simulate process flow, without really altering filesystem")
	argVersion = flag.Bool("version", false, "Print version")
	flag.Parse()

	if *argVersion {
		fmt.Println(fmt.Sprintf("SpotiTube, version %d.", spttb_system.Version))
		os.Exit(0)
	}

	if len(argFix.Paths) > 0 {
		*argReplaceLocal = true
		*argFlushMetadata = true
	}

	if *argManualInput {
		*argInteractive = true
	}

	if !(spttb_system.Dir(*argFolder)) {
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

	var guiOptions uint64
	if *argDebug {
		guiOptions = guiOptions | spttb_gui.GuiDebugMode
	}
	if *argDisableGui {
		guiOptions = guiOptions | spttb_gui.GuiSilentMode
	}
	gui = spttb_gui.Build(guiOptions)

	gui.Append(fmt.Sprintf("%s %s", spttb_gui.MessageStyle("Folder:", spttb_gui.FontStyleBold), *argFolder), spttb_gui.PanelLeftTop)
	if *argLog {
		gui.Append(fmt.Sprintf("%s %s", spttb_gui.MessageStyle("Log:", spttb_gui.FontStyleBold), spttb_logger.DefaultLogFname), spttb_gui.PanelLeftTop)
	}
	gui.Append(fmt.Sprintf("%s %d", spttb_gui.MessageStyle("Version:", spttb_gui.FontStyleBold), spttb_system.Version), spttb_gui.PanelLeftBottom)
	if os.Getenv("FORKED") == "1" {
		gui.Append(fmt.Sprintf("%s %s", spttb_gui.MessageStyle("Caller:", spttb_gui.FontStyleBold), "automatically updated"), spttb_gui.PanelLeftBottom)
	} else {
		gui.Append(fmt.Sprintf("%s %s", spttb_gui.MessageStyle("Caller:", spttb_gui.FontStyleBold), "installed"), spttb_gui.PanelLeftBottom)
	}
	gui.Append(fmt.Sprintf("%s %s", spttb_gui.MessageStyle("Date:", spttb_gui.FontStyleBold), time.Now().Format("2006-01-02 15:04:05")), spttb_gui.PanelLeftBottom)
	gui.Append(fmt.Sprintf("%s %s", spttb_gui.MessageStyle("URL:", spttb_gui.FontStyleBold), spttb_system.VersionRepository), spttb_gui.PanelLeftBottom)
	gui.Append(fmt.Sprintf("%s GPLv2", spttb_gui.MessageStyle("License:", spttb_gui.FontStyleBold)), spttb_gui.PanelLeftBottom)

	subLinkLog()
	subCheckDependencies()
	subCheckInternet()
	subCheckUpdate()

	if *argDisableGui {
		channel := make(chan os.Signal, 1)
		signal.Notify(channel, os.Interrupt)
		go func() {
			for range channel {
				subSafeExit()
			}
		}()
	} else {
		go func() {
			<-gui.Closing
			subSafeExit()
		}()
	}

	mainFetch()
}

func mainFetch() {
	if len(argFix.Paths) == 0 {
		spotifyAuthURL := spttb_spotify.AuthURL()
		gui.Append(fmt.Sprintf("Authentication URL: %s", spotifyAuthURL.Short), spttb_gui.PanelRight|spttb_gui.ParagraphStyleAutoReturn)
		gui.DebugAppend("Waiting for automatic login process. If wait is too long, manually open that URL.", spttb_gui.PanelRight)
		if !spotifyClient.Auth(spotifyAuthURL.Full) {
			gui.Prompt("Unable to authenticate to spotify.", spttb_gui.PromptDismissableWithExit)
		}
		gui.Append("Authentication completed.", spttb_gui.PanelRight)

		var (
			tracksOnline          []api.FullTrack
			tracksOnlineAlbums    []api.FullAlbum
			tracksOnlineAlbumsIds []api.ID
			tracksErr             error
		)
		if *argPlaylist == "none" {
			gui.Append("Fetching music library...", spttb_gui.PanelRight)
			if tracksOnline, tracksErr = spotifyClient.LibraryTracks(); tracksErr != nil {
				gui.Prompt(fmt.Sprintf("Something went wrong while fetching playlist: %s.", tracksErr.Error()), spttb_gui.PromptDismissableWithExit)
			}
		} else {
			gui.Append("Fetching playlist...", spttb_gui.PanelRight)
			var playlistErr error
			playlistInfo, playlistErr = spotifyClient.Playlist(*argPlaylist)
			if playlistErr != nil {
				gui.Prompt("Something went wrong while fetching playlist info.", spttb_gui.PromptDismissableWithExit)
			} else {
				gui.Append(fmt.Sprintf("%s %s", spttb_gui.MessageStyle("Playlist name:", spttb_gui.FontStyleBold), playlistInfo.Name), spttb_gui.PanelLeftTop)
				if len(playlistInfo.Owner.DisplayName) == 0 && len(strings.Split(*argPlaylist, ":")) >= 3 {
					gui.Append(fmt.Sprintf("%s %s", spttb_gui.MessageStyle("Playlist owner:", spttb_gui.FontStyleBold), strings.Split(*argPlaylist, ":")[2]), spttb_gui.PanelLeftTop)
				} else {
					gui.Append(fmt.Sprintf("%s %s", spttb_gui.MessageStyle("Playlist owner:", spttb_gui.FontStyleBold), playlistInfo.Owner.DisplayName), spttb_gui.PanelLeftTop)
				}
				gui.Append(fmt.Sprintf("Getting songs from \"%s\" playlist, by \"%s\"...", playlistInfo.Name, playlistInfo.Owner.DisplayName), spttb_gui.PanelRight|spttb_gui.FontStyleBold)
				if tracksOnline, tracksErr = spotifyClient.PlaylistTracks(*argPlaylist); tracksErr != nil {
					gui.Prompt(fmt.Sprintf("Something went wrong while fetching playlist: %s.", tracksErr.Error()), spttb_gui.PromptDismissableWithExit)
				}
			}
		}
		for _, track := range tracksOnline {
			tracksOnlineAlbumsIds = append(tracksOnlineAlbumsIds, track.Album.ID)
		}
		if tracksOnlineAlbums, tracksErr = spotifyClient.Albums(tracksOnlineAlbumsIds); tracksErr != nil {
			gui.Prompt(fmt.Sprintf("Something went wrong while fetching album info: %s.", tracksErr.Error()), spttb_gui.PromptDismissableWithExit)
		}

		gui.Append("Checking which songs need to be downloaded...", spttb_gui.PanelRight)
		var (
			tracksDuplicates = 0
			tracksMap        = make(map[string]float64)
		)
		for trackIndex := len(tracksOnline) - 1; trackIndex >= 0; trackIndex-- {
			if _, alreadyParsed := tracksMap[spttb_track.IDString(tracksOnline[trackIndex])]; !alreadyParsed {
				tracks = append(tracks, spttb_track.ParseSpotifyTrack(tracksOnline[trackIndex], tracksOnlineAlbums[trackIndex]))
				tracksMap[spttb_track.IDString(tracksOnline[trackIndex])] = 1
			} else {
				gui.WarnAppend(fmt.Sprintf("Ignored song duplicate \"%s\" by \"%s\".", tracksOnline[trackIndex].SimpleTrack.Name, tracksOnline[trackIndex].SimpleTrack.Artists[0].Name), spttb_gui.PanelRight)
				tracksDuplicates++
			}
		}

		gui.Append(fmt.Sprintf("%s %d", spttb_gui.MessageStyle("Songs online:", spttb_gui.FontStyleBold), len(tracks)), spttb_gui.PanelLeftTop)
		gui.Append(fmt.Sprintf("%s %d", spttb_gui.MessageStyle("Songs offline:", spttb_gui.FontStyleBold), tracks.CountOffline()), spttb_gui.PanelLeftTop)
		gui.Append(fmt.Sprintf("%s %d", spttb_gui.MessageStyle("Songs missing:", spttb_gui.FontStyleBold), tracks.CountOnline()), spttb_gui.PanelLeftTop)
		gui.Append(fmt.Sprintf("%s %d", spttb_gui.MessageStyle("Songs duplicates:", spttb_gui.FontStyleBold), tracksDuplicates), spttb_gui.PanelLeftTop)
	} else {
		gui.Append(fmt.Sprintf("%s %d", spttb_gui.MessageStyle("Fix song(s):", spttb_gui.FontStyleBold), len(argFix.Paths)), spttb_gui.PanelLeftTop)
		for _, fixTrack := range argFix.Paths {
			if track, trackErr := spttb_track.OpenLocalTrack(fixTrack); trackErr != nil {
				gui.Prompt(fmt.Sprintf("Something went wrong: %s.", trackErr.Error()), spttb_gui.PromptDismissableWithExit)
				mainExit()
			} else {
				gui.DebugAppend(fmt.Sprintf("%+v\n", track), spttb_gui.PanelRight)
				tracks = append(tracks, track)
			}
		}
	}

	for range [spttb_system.ConcurrencyLimit]int{} {
		waitGroupPool <- true
	}

	if len(tracks) > 0 {
		mainSearch()
	} else {
		gui.Prompt("No song needs to be downloaded.", spttb_gui.PromptDismissableWithExit)
		mainExit()
	}
}

func mainSearch() {
	defer mainExit()
	defer subCleanJunks()

	gui.LoadingSetMax(len(tracks))

	songsFetch, songsFlush, songsIgnore := subCountSongs()
	gui.Append(fmt.Sprintf("%d will be downloaded, %d flushed and %d ignored", songsFetch, songsFlush, songsIgnore), spttb_gui.PanelRight)

	for trackIndex, track := range tracks {
		gui.LoadingHalfIncrease()
		gui.Append(fmt.Sprintf("%d/%d: \"%s\"", trackIndex+1, len(tracks), track.Filename), spttb_gui.PanelRight|spttb_gui.FontStyleBold)
		if subIfSongSearch(track) {
			var (
				youTubeTrack         = spttb_youtube.Track{Track: &track}
				youTubeTracks        = spttb_youtube.Tracks{}
				youTubeTracksErr     error
				youTubeTrackPickAuto bool
				youTubeTrackPick     bool
			)
			if !*argManualInput {
				youTubeTracks, youTubeTracksErr = spttb_youtube.QueryTracks(&track)
				if youTubeTracksErr != nil {
					gui.WarnAppend(fmt.Sprintf("Something went wrong while searching for \"%s\" track: %s.", track.Filename, youTubeTracksErr.Error()), spttb_gui.PanelRight)
					tracksFailed = append(tracksFailed, track)
					gui.LoadingHalfIncrease()
					continue
				}
				for _, youTubeTrackLoopEl := range youTubeTracks {
					gui.DebugAppend(fmt.Sprintf("Result met: ID: %s,\nTitle: %s,\nUser: %s,\nDuration: %d.",
						youTubeTrackLoopEl.ID, youTubeTrackLoopEl.Title, youTubeTrackLoopEl.User, youTubeTrackLoopEl.Duration), spttb_gui.PanelRight)

					youTubeTrackPickAuto, youTubeTrackPick = subMatchResult(track, youTubeTrackLoopEl)
					if subIfPickFromAns(youTubeTrackPickAuto, youTubeTrackPick) {
						gui.Append(fmt.Sprintf("Video \"%s\" is good to go for \"%s\".", youTubeTrackLoopEl.Title, track.Filename), spttb_gui.PanelRight)
						youTubeTrack = youTubeTrackLoopEl
						break
					}
				}
			}

			subCondManualInputURL(&youTubeTrack)
			if youTubeTrack.URL == "" {
				youTubeTrack = spttb_youtube.Track{}
			} else {
				youTubeTrack.Track = &track
			}

			if youTubeTrack == (spttb_youtube.Track{}) {
				gui.ErrAppend(fmt.Sprintf("Video for \"%s\" not found.", track.Filename), spttb_gui.PanelRight)
				tracksFailed = append(tracksFailed, track)
				gui.LoadingHalfIncrease()
				continue
			}

			if *argSimulate {
				gui.Append(fmt.Sprintf("I would like to download \"%s\" for \"%s\" track, but I'm just simulating.", youTubeTrack.URL, track.Filename), spttb_gui.PanelRight)
				gui.LoadingHalfIncrease()
				continue
			} else if *argReplaceLocal {
				if track.URL == youTubeTrack.URL && !youTubeTrackPick {
					gui.Append(fmt.Sprintf("Track \"%s\" is still the best result I can find.", track.Filename), spttb_gui.PanelRight)
					gui.DebugAppend(fmt.Sprintf("Local track origin URL %s is the same as YouTube chosen one %s.", track.URL, youTubeTrack.URL), spttb_gui.PanelRight)
					gui.LoadingHalfIncrease()
					continue
				} else {
					track.URL = ""
					track.Local = false
				}
			}

			gui.Append(fmt.Sprintf("Going to download \"%s\" from %s...", youTubeTrack.Title, youTubeTrack.URL), spttb_gui.PanelRight)
			err := youTubeTrack.Download()
			if err != nil {
				gui.WarnAppend(fmt.Sprintf("Something went wrong downloading \"%s\": %s.", track.Filename, err.Error()), spttb_gui.PanelRight)
				tracksFailed = append(tracksFailed, track)
				gui.LoadingHalfIncrease()
				continue
			} else {
				track.URL = youTubeTrack.URL
			}
		}

		if !subIfSongProcess(track) {
			gui.LoadingHalfIncrease()
			continue
		}

		subCondSequentialDo(&track)

		gui.Append(fmt.Sprintf("Launching song processing jobs..."), spttb_gui.PanelRight)
		waitGroup.Add(1)
		go subParallelSongProcess(track, &waitGroup)
		if *argDebug {
			waitGroup.Wait()
		}
	}
	waitGroup.Wait()

	subCondPlaylistFileWrite()
	subCondTimestampFlush()

	junks := subCleanJunks()
	gui.Append(fmt.Sprintf("Removed %d junk files.", junks), spttb_gui.PanelRight)

	close(waitGroupPool)
	waitGroup.Wait()
	gui.LoadingFill()

	gui.Append(fmt.Sprintf("%d tracks failed to synchronize.", len(tracksFailed)), spttb_gui.PanelRight)
	for _, track := range tracksFailed {
		gui.Append(fmt.Sprintf(" - \"%s\"", track.Filename), spttb_gui.PanelRight)
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

	gui.Prompt("Synchronization completed.", spttb_gui.PromptDismissableWithExit)
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
			gui.Prompt(fmt.Sprintf("Are you sure %s is asctually installed?", commandName), spttb_gui.PromptDismissableWithExit)
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
				fmt.Println(commandOut.String())
				commandObj.Stdout = &commandOut
				commandObj.Stderr = &commandOut
				_ = commandObj.Run()
				if commandVersionRegValue := versionRegex.FindString(commandOut.String()); len(commandVersionRegValue) > 0 {
					commandVersionValue = commandVersionRegValue
				}
			}
			gui.Append(fmt.Sprintf("%s %s", spttb_gui.MessageStyle(fmt.Sprintf("Version %s:", commandName), spttb_gui.FontStyleBold), commandVersionValue), spttb_gui.PanelLeftTop)
		}
	}
}

func subCheckInternet() {
	_, err := net.Dial("tcp", spttb_system.TCPCheckOrigin)
	if err != nil {
		gui.Prompt("Are you sure you're connected to the internet?", spttb_gui.PromptDismissableWithExit)
	}
}

func subLinkLog() {
	if *argLog {
		err := gui.LinkLogger(spttb_logger.Build(spttb_logger.DefaultLogFname))
		if err != nil {
			gui.Prompt(fmt.Sprintf("Something went wrong while linking logger to %s", spttb_logger.DefaultLogFname), spttb_gui.PromptDismissableWithExit)
		}
	}
}

func subCheckUpdate() {
	if !*argDisableUpdateCheck {
		type OnlineVersion struct {
			Name string `json:"name"`
		}
		versionClient := http.Client{
			Timeout: time.Second * spttb_system.HTTPTimeout,
		}
		versionRequest, versionError := http.NewRequest(http.MethodGet, spttb_system.VersionOrigin, nil)
		if versionError != nil {
			gui.WarnAppend(fmt.Sprintf("Unable to compile version request: %s", versionError.Error()), spttb_gui.PanelRight)
		} else {
			versionResponse, versionError := versionClient.Do(versionRequest)
			if versionError != nil {
				gui.WarnAppend(fmt.Sprintf("Unable to read response from version request: %s", versionError.Error()), spttb_gui.PanelRight)
			} else {
				versionResponseBody, versionError := ioutil.ReadAll(versionResponse.Body)
				if versionError != nil {
					gui.WarnAppend(fmt.Sprintf("Unable to get response body: %s", versionError.Error()), spttb_gui.PanelRight)
				} else {
					versionData := OnlineVersion{}
					versionError = json.Unmarshal(versionResponseBody, &versionData)
					if versionError != nil {
						gui.WarnAppend(fmt.Sprintf("Unable to parse json from response body: %s", versionError.Error()), spttb_gui.PanelRight)
					} else {
						versionValue := 0
						versionRegex, versionError := regexp.Compile("[^0-9]+")
						if versionError != nil {
							gui.WarnAppend(fmt.Sprintf("Unable to compile regex needed to parse version: %s", versionError.Error()), spttb_gui.PanelRight)
						} else {
							versionValue, versionError = strconv.Atoi(versionRegex.ReplaceAllString(versionData.Name, ""))
							if versionError != nil {
								gui.WarnAppend(fmt.Sprintf("Unable to fetch latest version value: %s", versionError.Error()), spttb_gui.PanelRight)
							} else if versionValue != spttb_system.Version {
								gui.Append(fmt.Sprintf("Going to update from %d to %d version.", spttb_system.Version, versionValue), spttb_gui.PanelRight)
								subUpdateSoftware(versionResponseBody)
							}
							gui.DebugAppend(fmt.Sprintf("Actual version %d, online version %d.", spttb_system.Version, versionValue), spttb_gui.PanelRight)
						}
					}
				}
			}
		}
	}
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
		gui.WarnAppend(fmt.Sprintf("Unable to unmarshal Github latest relase data: %s", unmarshalErr.Error()), spttb_gui.PanelRight)
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
		gui.WarnAppend(fmt.Sprintf("Unable to create temporary updated binary file: %s", err.Error()), spttb_gui.PanelRight)
		return
	}
	defer binaryOutput.Close()

	gui.Append(fmt.Sprintf("Downloading update from %s...", binaryURL), spttb_gui.PanelRight)
	binaryPayload, err := http.Get(binaryURL)
	if err != nil {
		gui.WarnAppend(fmt.Sprintf("Unable to download from %s: %s", binaryURL, err.Error()), spttb_gui.PanelRight)
		return
	}
	defer binaryPayload.Body.Close()

	_, err = io.Copy(binaryOutput, binaryPayload.Body)
	if err != nil {
		gui.WarnAppend(fmt.Sprintf("Error while downloading from %s: %s", binaryURL, err.Error()), spttb_gui.PanelRight)
		return
	}

	if user, err := user.Current(); err == nil {
		var (
			binaryFolder string = fmt.Sprintf("%s/.spotitube", user.HomeDir)
			binaryFname  string = fmt.Sprintf("%s/spotitube", binaryFolder)
		)
		err = spttb_system.Mkdir(binaryFolder)
		if err != nil {
			gui.WarnAppend(fmt.Sprintf("Unable to create binary container folder at %s: %s", binaryFolder, err.Error()), spttb_gui.PanelRight)
			return
		}
		os.Remove(binaryFname)
		err = spttb_system.FileCopy(binaryTempFname, binaryFname)
		if err != nil {
			gui.WarnAppend(fmt.Sprintf("Unable to persist new binary to %s: %s", binaryFname, err.Error()), spttb_gui.PanelRight)
			return
		} else {
			os.Remove(binaryTempFname)
		}

		err = os.Chmod(binaryFname, 0755)
		if err != nil {
			gui.WarnAppend(fmt.Sprintf("Unable to make %s executable: %s", binaryFname, err.Error()), spttb_gui.PanelRight)
			return
		}

		err = syscall.Exec(binaryFname, os.Args, os.Environ())
		if err != nil {
			gui.ErrAppend(fmt.Sprintf("Unable to exec updated instance: %s", err.Error()), spttb_gui.PanelRight)
		}
		mainExit()
	} else {
		return
	}
}

func subParallelSongProcess(track spttb_track.Track, wg *sync.WaitGroup) {
	defer gui.LoadingHalfIncrease()
	defer wg.Done()
	<-waitGroupPool

	if !track.Local && !*argDisableNormalization {
		subSongNormalize(track)
	}

	if !spttb_system.FileExists(track.FilenameTemporary()) && spttb_system.FileExists(track.FilenameFinal()) {
		if err := spttb_system.FileCopy(track.FilenameFinal(), track.FilenameTemporary()); err != nil {
			gui.WarnAppend(fmt.Sprintf("Unable to prepare song for getting its metadata flushed: %s", err.Error()), spttb_gui.PanelRight)
			return
		}
	}

	if (track.Local && *argFlushMetadata) || !track.Local {
		subSongFlushMetadata(track)
	}

	os.Remove(track.FilenameFinal())
	err := os.Rename(track.FilenameTemporary(), track.FilenameFinal())
	if err != nil {
		gui.WarnAppend(fmt.Sprintf("Unable to move song to its final path: %s", err.Error()), spttb_gui.PanelRight)
	}

	waitGroupPool <- true
}

func subSongNormalize(track spttb_track.Track) {
	var (
		commandCmd         = "ffmpeg"
		commandArgs        []string
		commandOut         bytes.Buffer
		commandErr         error
		normalizationDelta string
		normalizationFile  = strings.Replace(track.FilenameTemporary(), track.FilenameExt, ".norm"+track.FilenameExt, -1)
	)

	commandArgs = []string{"-i", track.FilenameTemporary(), "-af", "volumedetect", "-f", "null", "-y", "null"}
	gui.DebugAppend(fmt.Sprintf("Getting max_volume value: \"%s %s\"...", commandCmd, strings.Join(commandArgs, " ")), spttb_gui.PanelRight)
	commandObj := exec.Command(commandCmd, commandArgs...)
	commandObj.Stderr = &commandOut
	commandErr = commandObj.Run()
	if commandErr != nil {
		gui.WarnAppend(fmt.Sprintf("Unable to use ffmpeg to pull max_volume song value: %s.", commandOut.String()), spttb_gui.PanelRight)
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
		gui.WarnAppend(fmt.Sprintf("Unable to pull max_volume delta to be applied along with song volume normalization: %s.", normalizationDelta), spttb_gui.PanelRight)
		normalizationDelta = "0.0"
	}
	commandArgs = []string{"-i", track.FilenameTemporary(), "-af", "volume=+" + normalizationDelta + "dB", "-b:a", "320k", "-y", normalizationFile}
	gui.DebugAppend(fmt.Sprintf("Compensating volume by %sdB...", normalizationDelta), spttb_gui.PanelRight)
	gui.DebugAppend(fmt.Sprintf("Increasing audio quality for: %s...", track.Filename), spttb_gui.PanelRight)
	gui.DebugAppend(fmt.Sprintf("Firing command: \"%s %s\"...", commandCmd, strings.Join(commandArgs, " ")), spttb_gui.PanelRight)
	if _, commandErr = exec.Command(commandCmd, commandArgs...).Output(); commandErr != nil {
		gui.WarnAppend(fmt.Sprintf("Something went wrong while normalizing song \"%s\" volume: %s", track.Filename, commandErr.Error()), spttb_gui.PanelRight)
	}
	os.Remove(track.FilenameTemporary())
	os.Rename(normalizationFile, track.FilenameTemporary())
}

func subSongFlushMetadata(track spttb_track.Track) {
	trackMp3, err := id3.Open(track.FilenameTemporary(), id3.Options{Parse: true})
	if err != nil {
		gui.WarnAppend(fmt.Sprintf("Something bad happened while opening: %s", err.Error()), spttb_gui.PanelRight)
	} else {
		gui.DebugAppend(fmt.Sprintf("Fixing metadata for \"%s\"...", track.Filename), spttb_gui.PanelRight)
		if !*argFlushMissing && !*argFlushDifferent {
			trackMp3.DeleteAllFrames()
		}
		subCondFlushID3FrameTitle(track, trackMp3)
		subCondFlushID3FrameSong(track, trackMp3)
		subCondFlushID3FrameArtist(track, trackMp3)
		subCondFlushID3FrameAlbum(track, trackMp3)
		subCondFlushID3FrameGenre(track, trackMp3)
		subCondFlushID3FrameYear(track, trackMp3)
		subCondFlushID3FrameFeaturings(track, trackMp3)
		subCondFlushID3FrameTrackNumber(track, trackMp3)
		subCondFlushID3FrameTrackTotals(track, trackMp3)
		subCondFlushID3FrameArtwork(track, trackMp3)
		subCondFlushID3FrameArtworkURL(track, trackMp3)
		subCondFlushID3FrameYouTubeURL(track, trackMp3)
		subCondFlushID3FrameDuration(track, trackMp3)
		subCondFlushID3FrameLyrics(track, trackMp3)
		trackMp3.Save()
	}
	trackMp3.Close()
}

func subCondFlushID3FrameTitle(track spttb_track.Track, trackMp3 *id3.Tag) {
	if len(track.Title) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameTitle))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameTitle) != track.Title)) {
		gui.DebugAppend("Inflating title metadata...", spttb_gui.PanelRight)
		trackMp3.SetTitle(track.Title)
	}
}

func subCondFlushID3FrameSong(track spttb_track.Track, trackMp3 *id3.Tag) {
	if len(track.Song) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameSong))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameSong) != track.Song)) {
		gui.DebugAppend("Inflating song metadata...", spttb_gui.PanelRight)
		trackMp3.AddCommentFrame(id3.CommentFrame{
			Encoding:    id3.EncodingUTF8,
			Language:    "eng",
			Description: "song",
			Text:        track.Song,
		})
	}
}

func subCondFlushID3FrameArtist(track spttb_track.Track, trackMp3 *id3.Tag) {
	if len(track.Artist) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameArtist))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameArtist) != track.Artist)) {
		gui.DebugAppend("Inflating artist metadata...", spttb_gui.PanelRight)
		trackMp3.SetArtist(track.Artist)
	}
}

func subCondFlushID3FrameAlbum(track spttb_track.Track, trackMp3 *id3.Tag) {
	if len(track.Album) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameAlbum))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameAlbum) != track.Album)) {
		gui.DebugAppend("Inflating album metadata...", spttb_gui.PanelRight)
		trackMp3.SetAlbum(track.Album)
	}
}

func subCondFlushID3FrameGenre(track spttb_track.Track, trackMp3 *id3.Tag) {
	if len(track.Genre) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameGenre))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameGenre) != track.Genre)) {
		gui.DebugAppend("Inflating genre metadata...", spttb_gui.PanelRight)
		trackMp3.SetGenre(track.Genre)
	}
}

func subCondFlushID3FrameYear(track spttb_track.Track, trackMp3 *id3.Tag) {
	if len(track.Year) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameYear))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameYear) != track.Year)) {
		gui.DebugAppend("Inflating year metadata...", spttb_gui.PanelRight)
		trackMp3.SetYear(track.Year)
	}
}

func subCondFlushID3FrameFeaturings(track spttb_track.Track, trackMp3 *id3.Tag) {
	if len(track.Featurings) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameFeaturings))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameFeaturings) != strings.Join(track.Featurings, "|"))) {
		gui.DebugAppend("Inflating featurings metadata...", spttb_gui.PanelRight)
		trackMp3.AddCommentFrame(id3.CommentFrame{
			Encoding:    id3.EncodingUTF8,
			Language:    "eng",
			Description: "featurings",
			Text:        strings.Join(track.Featurings, "|"),
		})
	}
}

func subCondFlushID3FrameTrackNumber(track spttb_track.Track, trackMp3 *id3.Tag) {
	if track.TrackNumber > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameTrackNumber))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameTrackNumber) != fmt.Sprintf("%d", track.TrackNumber))) {
		gui.DebugAppend("Inflating track number metadata...", spttb_gui.PanelRight)
		trackMp3.AddFrame(trackMp3.CommonID("Track number/Position in set"),
			id3.TextFrame{
				Encoding: id3.EncodingUTF8,
				Text:     strconv.Itoa(track.TrackNumber),
			})
	}
}

func subCondFlushID3FrameTrackTotals(track spttb_track.Track, trackMp3 *id3.Tag) {
	if track.TrackTotals > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameTrackTotals))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameTrackTotals) != fmt.Sprintf("%d", track.TrackTotals))) {
		gui.DebugAppend("Inflating total tracks number metadata...", spttb_gui.PanelRight)
		trackMp3.AddCommentFrame(id3.CommentFrame{
			Encoding:    id3.EncodingUTF8,
			Language:    "eng",
			Description: "trackTotals",
			Text:        fmt.Sprintf("%d", track.TrackTotals),
		})
	}
}

func subCondFlushID3FrameArtwork(track spttb_track.Track, trackMp3 *id3.Tag) {
	if spttb_system.FileExists(track.FilenameArtwork()) &&
		(!*argFlushMissing || (*argFlushMissing && !track.HasID3Frame(spttb_track.ID3FrameArtwork))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameArtworkURL) != track.Image)) {
		trackArtworkReader, trackArtworkErr := ioutil.ReadFile(track.FilenameArtwork())
		if trackArtworkErr != nil {
			gui.WarnAppend(fmt.Sprintf("Unable to read artwork file: %s", trackArtworkErr.Error()), spttb_gui.PanelRight)
		} else {
			gui.DebugAppend("Inflating artwork metadata...", spttb_gui.PanelRight)
			trackMp3.AddAttachedPicture(id3.PictureFrame{
				Encoding:    id3.EncodingUTF8,
				MimeType:    "image/jpeg",
				PictureType: id3.PTFrontCover,
				Description: "Front cover",
				Picture:     trackArtworkReader,
			})
		}
	}
}

func subCondFlushID3FrameArtworkURL(track spttb_track.Track, trackMp3 *id3.Tag) {
	if len(track.Image) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameArtworkURL))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameArtworkURL) != track.Image)) {
		gui.DebugAppend("Inflating artwork url metadata...", spttb_gui.PanelRight)
		trackMp3.AddCommentFrame(id3.CommentFrame{
			Encoding:    id3.EncodingUTF8,
			Language:    "eng",
			Description: "artwork",
			Text:        track.Image,
		})
	}
}

func subCondFlushID3FrameYouTubeURL(track spttb_track.Track, trackMp3 *id3.Tag) {
	if len(track.URL) > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameYouTubeURL))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameYouTubeURL) != track.URL)) {
		gui.DebugAppend("Inflating youtube origin url metadata...", spttb_gui.PanelRight)
		trackMp3.AddCommentFrame(id3.CommentFrame{
			Encoding:    id3.EncodingUTF8,
			Language:    "eng",
			Description: "youtube",
			Text:        track.URL,
		})
	}
}

func subCondFlushID3FrameDuration(track spttb_track.Track, trackMp3 *id3.Tag) {
	if track.Duration > 0 &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameDuration))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameDuration) != fmt.Sprintf("%d", track.Duration))) {
		gui.DebugAppend("Inflating duration metadata...", spttb_gui.PanelRight)
		trackMp3.AddCommentFrame(id3.CommentFrame{
			Encoding:    id3.EncodingUTF8,
			Language:    "eng",
			Description: "duration",
			Text:        fmt.Sprintf("%d", track.Duration),
		})
	}
}

func subCondFlushID3FrameLyrics(track spttb_track.Track, trackMp3 *id3.Tag) {
	if len(track.Lyrics) > 0 && !*argDisableLyrics &&
		(!*argFlushMissing || (*argFlushMissing && !spttb_track.TagHasFrame(trackMp3, spttb_track.ID3FrameLyrics))) &&
		(!*argFlushDifferent || (*argFlushDifferent && spttb_track.TagGetFrame(trackMp3, spttb_track.ID3FrameLyrics) != track.Lyrics)) {
		gui.DebugAppend("Inflating lyrics metadata...", spttb_gui.PanelRight)
		trackMp3.AddUnsynchronisedLyricsFrame(id3.UnsynchronisedLyricsFrame{
			Encoding:          id3.EncodingUTF8,
			Language:          "eng",
			ContentDescriptor: track.Title,
			Lyrics:            track.Lyrics,
		})
	}
}

func subIfSongSearch(track spttb_track.Track) bool {
	return !track.Local || *argReplaceLocal || *argSimulate
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

func subMatchResult(track spttb_track.Track, youTubeTrack spttb_youtube.Track) (bool, bool) {
	var (
		ansInput     bool
		ansAutomated bool
		ansErr       error
	)
	ansErr = youTubeTrack.Match(track)
	ansAutomated = bool(ansErr == nil)
	if *argInteractive {
		ansInput = gui.PromptInput(fmt.Sprintf("Do you want to download the following video for \"%s\"?\n"+
			"ID: %s\nTitle: %s\nUser: %s\nDuration: %d\nURL: %s\nScore: %d/6\nResult is matching: %s",
			track.Filename, youTubeTrack.ID, youTubeTrack.Title, youTubeTrack.User,
			youTubeTrack.Duration, youTubeTrack.URL, youTubeTrack.AffinityScore,
			strconv.FormatBool(ansAutomated)), spttb_gui.OptionNil)
	}
	return ansAutomated, ansInput
}

func subIfPickFromAns(ansAutomated bool, ansInput bool) bool {
	return (!*argInteractive && ansAutomated) || (*argInteractive && ansInput)
}

func subCondManualInputURL(youTubeTrack *spttb_youtube.Track) {
	if *argInteractive && youTubeTrack.URL == "" {
		inputURL := gui.PromptInputMessage(fmt.Sprintf("Please, manually enter URL for \"%s\"", youTubeTrack.Track.Filename), spttb_gui.PromptDismissable)
		if len(inputURL) > 0 {
			if err := spttb_youtube.ValidateURL(inputURL); err == nil {
				youTubeTrack.Title = "input video"
				youTubeTrack.URL = inputURL
			} else {
				gui.Prompt(fmt.Sprintf("Something went wrong: %s", err.Error()), spttb_gui.PromptDismissable)
			}
		}
	}
}

func subIfSongProcess(track spttb_track.Track) bool {
	return !track.Local || *argFlushMetadata || *argReplaceLocal
}

func subCondSequentialDo(track *spttb_track.Track) {
	if (track.Local && *argFlushMetadata) || !track.Local {
		subCondLyricsFetch(track)
		subCondArtworkDownload(track)
	}
}

func subCondLyricsFetch(track *spttb_track.Track) {
	if !*argDisableLyrics &&
		(!*argFlushMissing || (*argFlushMissing && !track.HasID3Frame(spttb_track.ID3FrameLyrics))) {
		gui.DebugAppend(fmt.Sprintf("Fetching song \"%s\" lyrics...", track.Filename), spttb_gui.PanelRight)
		lyricsErr := track.SearchLyrics()
		if lyricsErr != nil {
			gui.WarnAppend(fmt.Sprintf("Something went wrong while searching for song lyrics: %s", lyricsErr.Error()), spttb_gui.PanelRight)
		} else {
			gui.DebugAppend(fmt.Sprintf("Song lyrics found."), spttb_gui.PanelRight)
		}
	}
}

func subCondArtworkDownload(track *spttb_track.Track) {
	if !spttb_system.FileExists(track.FilenameArtwork()) &&
		(!*argFlushMissing || (*argFlushMissing && !track.HasID3Frame(spttb_track.ID3FrameArtwork))) {
		gui.DebugAppend(fmt.Sprintf("Downloading song \"%s\" artwork at %s...", track.Filename, track.Image), spttb_gui.PanelRight)
		var commandOut bytes.Buffer
		commandCmd := "ffmpeg"
		commandArgs := []string{"-i", track.Image, "-q:v", "1", "-n", track.FilenameArtwork()}
		commandObj := exec.Command(commandCmd, commandArgs...)
		commandObj.Stderr = &commandOut
		if err := commandObj.Run(); err != nil {
			gui.WarnAppend(fmt.Sprintf("Unable to download artwork file \"%s\": %s", track.Image, commandOut.String()), spttb_gui.PanelRight)
		}
	}
}

func subCondTimestampFlush() {
	if !*argDisableTimestampFlush {
		gui.Append("Flushing files timestamps...", spttb_gui.PanelRight)
		now := time.Now().Local().Add(time.Duration(-1*len(tracks)) * time.Minute)
		for _, track := range tracks {
			if !spttb_system.FileExists(track.FilenameFinal()) {
				continue
			}
			if err := os.Chtimes(track.FilenameFinal(), now, now); err != nil {
				gui.WarnAppend(fmt.Sprintf("Unable to flush timestamp on %s", track.FilenameFinal()), spttb_gui.PanelRight)
			}
			now = now.Add(1 * time.Minute)
		}
	}
}

func subCondPlaylistFileWrite() {
	if !*argSimulate && !*argDisablePlaylistFile && *argPlaylist != "none" {
		var (
			playlistFolder  = sanitize.Name(playlistInfo.Name)
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
		for _, track := range tracks {
			if spttb_system.FileExists("../" + track.FilenameFinal()) {
				if err := os.Symlink("../"+track.FilenameFinal(), track.FilenameFinal()); err != nil {
					gui.ErrAppend(fmt.Sprintf("Unable to create symlink for \"%s\" in %s: %s", track.FilenameFinal(), playlistFolder, err.Error()), spttb_gui.PanelRight)
				}
			}
		}
		os.Chdir("..")

		gui.Append(fmt.Sprintf("Creating playlist file at %s...", playlistFname), spttb_gui.PanelRight)
		if spttb_system.FileExists(playlistFname) {
			os.Remove(playlistFname)
		}

		if !*argPlsFile {
			playlistContent = "#EXTM3U\n"
			for trackIndex := len(tracks) - 1; trackIndex >= 0; trackIndex-- {
				track := tracks[trackIndex]
				if spttb_system.FileExists(track.FilenameFinal()) {
					playlistContent += "#EXTINF:" + strconv.Itoa(track.Duration) + "," + track.Filename + "\n" +
						"./" + track.FilenameFinal() + "\n"
				}
			}
		} else {
			gui.Append("Creating playlist PLS file...", spttb_gui.PanelRight)
			if spttb_system.FileExists(playlistInfo.Name + ".pls") {
				os.Remove(playlistInfo.Name + ".pls")
			}
			playlistContent = "[" + playlistInfo.Name + "]\n"
			for trackIndex := len(tracks) - 1; trackIndex >= 0; trackIndex-- {
				track := tracks[trackIndex]
				trackInvertedIndex := len(tracks) - trackIndex
				if spttb_system.FileExists(track.FilenameFinal()) {
					playlistContent += "File" + strconv.Itoa(trackInvertedIndex) + "=./" + track.FilenameFinal() + "\n" +
						"Title" + strconv.Itoa(trackInvertedIndex) + "=" + track.Filename + "\n" +
						"Length" + strconv.Itoa(trackInvertedIndex) + "=" + strconv.Itoa(track.Duration) + "\n\n"
				}
			}
			playlistContent += "NumberOfEntries=" + strconv.Itoa(len(tracks)) + "\n"
		}

		playlistFile, playlistErr := os.Create(playlistFname)
		if playlistErr != nil {
			gui.WarnAppend(fmt.Sprintf("Unable to create M3U file: %s", playlistErr.Error()), spttb_gui.PanelRight)
		} else {
			defer playlistFile.Close()
			_, playlistErr := playlistFile.WriteString(playlistContent)
			playlistFile.Sync()
			if playlistErr != nil {
				gui.WarnAppend(fmt.Sprintf("Unable to write M3U file: %s", playlistErr.Error()), spttb_gui.PanelRight)
			}
		}
	}
}

func subCleanJunks() int {
	var removedJunks int
	for _, junkType := range spttb_track.JunkWildcards() {
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
