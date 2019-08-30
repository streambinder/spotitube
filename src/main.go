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
	"./provider"
	"./spotify"
	"./spotitube"
	"./system"
	"./track"

	"github.com/0xAX/notificator"
	"github.com/bogem/id3v2"
	"github.com/gosimple/slug"
)

var (
	argFolder                string
	argPlaylist              string
	argInvalidateCache       bool
	argReplaceLocal          bool
	argFlushMetadata         bool
	argFlushMissing          bool
	argFlushDifferent        bool
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

	ui     *cui.CUI
	notify *notificator.Notificator

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

	flag.StringVar(&argFolder, "folder", ".", "Folder to sync with music")
	flag.StringVar(&argPlaylist, "playlist", "none", "Playlist URI to synchronize")
	flag.BoolVar(&argInvalidateCache, "invalidate-cache", false, "Manually invalidate library cache, retriggering its fetch from Spotify")
	flag.BoolVar(&argReplaceLocal, "replace-local", false, "Replace local library songs if better results get encountered")
	flag.BoolVar(&argFlushMetadata, "flush-metadata", false, "Flush metadata informations to already synchronized songs")
	flag.BoolVar(&argFlushMissing, "flush-missing", false, "If -flush-metadata toggled, it will just populate empty id3v2 frames, instead of flushing any of those")
	flag.BoolVar(&argFlushDifferent, "flush-different", false, "If -flush-metadata toggled, it will just populate id3v2 frames different from the ones calculated by the application, instead of flushing any of those")
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
	if runtime.GOOS != "windows" {
		flag.BoolVar(&argDisableGui, "disable-gui", false, "Disable GUI to reduce noise and increase readability of program flow")
	} else {
		argDisableGui = true
	}
	flag.Parse()

	if argVersion {
		fmt.Println(fmt.Sprintf("SpotiTube, version %d.", spotitube.Version))
		os.Exit(0)
	}

	if len(argFix.Paths) > 0 {
		argReplaceLocal = true
		argFlushMetadata = true
	}

	if argManualInput {
		argInteractive = true
	}

	if !(system.Dir(argFolder)) {
		fmt.Println(fmt.Sprintf("Chosen music folder does not exist: %s", argFolder))
		os.Exit(1)
	} else {
		argFolder, _ = filepath.Abs(argFolder)
		os.Chdir(argFolder)
		if user, err := user.Current(); err == nil {
			argFolder = strings.Replace(argFolder, user.HomeDir, "~", -1)
		}
	}

	if argCleanJunks {
		junks := subCleanJunks()
		fmt.Println(fmt.Sprintf("Removed %d junk files.", junks))
		os.Exit(0)
	}

	system.Mkdir(userLocalConfigPath)

	var guiOptions uint64
	if argDebug {
		guiOptions |= cui.GuiDebugMode
	}
	if argDisableGui {
		guiOptions |= cui.GuiBareMode
	}
	if argLog {
		guiOptions |= cui.LogEnable
	}

	var err error
	if ui, err = cui.Startup(guiOptions); err != nil {
		fmt.Println(fmt.Sprintf("Unable to build user interface: %s", err.Error()))
	}

	ui.Append(fmt.Sprintf("%s %s", cui.Font("Folder:", cui.StyleBold), argFolder), cui.PanelLeftTop)
	if argLog {
		ui.Append(fmt.Sprintf("%s %s", cui.Font("Log:", cui.StyleBold), logger.LogFilename), cui.PanelLeftTop)
	}
	ui.Append(fmt.Sprintf("%s %d", cui.Font("Version:", cui.StyleBold), spotitube.Version), cui.PanelLeftBottom)
	if os.Getenv("FORKED") == "1" {
		ui.Append(fmt.Sprintf("%s %s", cui.Font("Caller:", cui.StyleBold), "automatically updated"), cui.PanelLeftBottom)
	} else {
		ui.Append(fmt.Sprintf("%s %s", cui.Font("Caller:", cui.StyleBold), "installed"), cui.PanelLeftBottom)
	}
	ui.Append(fmt.Sprintf("%s %s", cui.Font("Date:", cui.StyleBold), time.Now().Format("2006-01-02 15:04:05")), cui.PanelLeftBottom)
	ui.Append(fmt.Sprintf("%s %s", cui.Font("URL:", cui.StyleBold), spotitube.VersionRepository), cui.PanelLeftBottom)
	ui.Append(fmt.Sprintf("%s GPL-3.0", cui.Font("License:", cui.StyleBold)), cui.PanelLeftBottom)

	subCheckDependencies()
	subCheckInternet()
	subCheckUpdate()
	subFetchIndex()

	if !argDisableIndexing {
		go subAlignIndex()
	}

	go func() {
		<-ui.CloseChan
		subSafeExit()
	}()
	if argDisableGui {
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
		if spotifyClient, err = spotify.Auth(spotifyAuthURL.Full, spotifyAuthHost, !argDisableBrowserOpening); err != nil {
			ui.Prompt("Unable to authenticate to spotify.", cui.PromptExit)
		}
		ui.Append("Authentication completed.")
		spotifyUser, spotifyUserID = spotifyClient.User()
		ui.Append(fmt.Sprintf("%s %s", cui.Font("Session user:", cui.StyleBold), spotifyUser), cui.PanelLeftTop)

		var (
			tracksOnline          []spotify.Track
			tracksOnlineAlbums    []spotify.Album
			tracksOnlineAlbumsIds []spotify.ID
			tracksErr             error
		)

		if argPlaylist == "none" {
			userLocalGob = fmt.Sprintf(userLocalGob, spotifyUserID, "library")
			if argInvalidateCache {
				os.Remove(userLocalGob)
			}
			tracksDump, tracksDumpErr := subFetchGob(userLocalGob)
			if tracksDumpErr != nil {
				ui.Append(tracksDumpErr.Error(), cui.WarningAppend)
				ui.Append("Fetching music library...")
				if tracksOnline, tracksErr = spotifyClient.LibraryTracks(); tracksErr != nil {
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
			playlistInfo, playlistErr = spotifyClient.Playlist(argPlaylist)
			if playlistErr != nil {
				ui.Prompt("Something went wrong while fetching playlist info.", cui.PromptExit)
			} else {
				ui.Append(fmt.Sprintf("%s %s", cui.Font("Playlist name:", cui.StyleBold), playlistInfo.Name), cui.PanelLeftTop)
				if len(playlistInfo.Owner.DisplayName) == 0 && len(strings.Split(argPlaylist, ":")) >= 3 {
					ui.Append(fmt.Sprintf("%s %s", cui.Font("Playlist owner:", cui.StyleBold), strings.Split(argPlaylist, ":")[2]), cui.PanelLeftTop)
				} else {
					ui.Append(fmt.Sprintf("%s %s", cui.Font("Playlist owner:", cui.StyleBold), playlistInfo.Owner.DisplayName), cui.PanelLeftTop)
				}

				userLocalGob = fmt.Sprintf(userLocalGob, playlistInfo.Owner.ID, playlistInfo.Name)
				if argInvalidateCache {
					os.Remove(userLocalGob)
				}
				tracksDump, tracksDumpErr := subFetchGob(userLocalGob)
				if tracksDumpErr != nil {
					ui.Append(tracksDumpErr.Error(), cui.WarningAppend)
					ui.Append(fmt.Sprintf("Getting songs from \"%s\" playlist, by \"%s\"...", playlistInfo.Name, playlistInfo.Owner.DisplayName), cui.StyleBold)
					if tracksOnline, tracksErr = spotifyClient.PlaylistTracks(argPlaylist); tracksErr != nil {
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
		if tracksOnlineAlbums, tracksErr = spotifyClient.Albums(tracksOnlineAlbumsIds); tracksErr != nil {
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
				if removeErr := spotifyClient.RemoveLibraryTracks(tracksDuplicates); removeErr != nil {
					ui.Prompt(fmt.Sprintf("Something went wrong while removing %d duplicates: %s.", len(tracksDuplicates), removeErr.Error()))
				} else {
					ui.Append(fmt.Sprintf("%d duplicate tracks correctly removed from library.", len(tracksDuplicates)))
				}
			} else {
				if removeErr := spotifyClient.RemovePlaylistTracks(argPlaylist, tracksDuplicates); removeErr != nil {
					ui.Prompt(fmt.Sprintf("Something went wrong while removing %d duplicates: %s.", len(tracksDuplicates), removeErr.Error()))
				} else {
					ui.Append(fmt.Sprintf("%d duplicate tracks correctly removed from playlist.", len(tracksDuplicates)))
				}
			}
		}

		if dumpErr := system.DumpGob(userLocalGob, track.TracksDump{Tracks: tracks, Time: time.Now()}); dumpErr != nil {
			ui.Append(fmt.Sprintf("Unable to cache tracks: %s", dumpErr.Error()), cui.WarningAppend)
		}

		ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs duplicates:", cui.StyleBold), len(tracksDuplicates)), cui.PanelLeftTop)
		ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs online:", cui.StyleBold), len(tracks)), cui.PanelLeftTop)
		ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs offline:", cui.StyleBold), tracks.CountOffline()), cui.PanelLeftTop)
		ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs missing:", cui.StyleBold), tracks.CountOnline()), cui.PanelLeftTop)

		<-waitIndex
		close(waitIndex)
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
	defer subCleanJunks()

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
				} else if argReplaceLocal {
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

	junks := subCleanJunks()
	ui.Append(fmt.Sprintf("Removed %d junk files.", junks))

	close(waitGroupPool)
	waitGroup.Wait()
	ui.ProgressFill()

	ui.Append(fmt.Sprintf("%d tracks failed to synchronize.", len(tracksFailed)))
	for _, t := range tracksFailed {
		ui.Append(fmt.Sprintf(" - \"%s\"", t.Basename()))
	}

	var (
		notify = notificator.New(notificator.Options{
			DefaultIcon: "emblem-downloads",
			AppName:     "SpotiTube",
		})
		notifyTitle   string
		notifyContent string
	)
	if argPlaylist == "none" {
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

	ui.Prompt("Synchronization completed.", cui.PromptExit)
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
			ui.Prompt(fmt.Sprintf("Are you sure %s is asctually installed?", commandName), cui.PromptExit)
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
			ui.Append(fmt.Sprintf("%s %s", cui.Font(fmt.Sprintf("Version %s:", commandName), cui.StyleBold), commandVersionValue), cui.PanelLeftTop)
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
		ui.Prompt("Are you sure you're connected to the internet?", cui.PromptExit)
	}
}

func subCheckUpdate() {
	if !argDisableUpdateCheck {
		type OnlineVersion struct {
			Name string `json:"name"`
		}
		versionClient := http.Client{
			Timeout: time.Second * spotitube.HTTPTimeout,
		}
		versionRequest, versionError := http.NewRequest(http.MethodGet, spotitube.VersionOrigin, nil)
		if versionError != nil {
			ui.Append(fmt.Sprintf("Unable to compile version request: %s", versionError.Error()), cui.WarningAppend)
		} else {
			versionResponse, versionError := versionClient.Do(versionRequest)
			if versionError != nil {
				ui.Append(fmt.Sprintf("Unable to read response from version request: %s", versionError.Error()), cui.WarningAppend)
			} else {
				versionResponseBody, versionError := ioutil.ReadAll(versionResponse.Body)
				if versionError != nil {
					ui.Append(fmt.Sprintf("Unable to get response body: %s", versionError.Error()), cui.WarningAppend)
				} else {
					versionData := OnlineVersion{}
					versionError = json.Unmarshal(versionResponseBody, &versionData)
					if versionError != nil {
						ui.Append(fmt.Sprintf("Unable to parse json from response body: %s", versionError.Error()), cui.WarningAppend)
					} else {
						versionValue := 0
						versionRegex, versionError := regexp.Compile("[^0-9]+")
						if versionError != nil {
							ui.Append(fmt.Sprintf("Unable to compile regex needed to parse version: %s", versionError.Error()), cui.WarningAppend)
						} else {
							versionValue, versionError = strconv.Atoi(versionRegex.ReplaceAllString(versionData.Name, ""))
							if versionError != nil {
								ui.Append(fmt.Sprintf("Unable to fetch latest version value: %s", versionError.Error()), cui.WarningAppend)
							} else if versionValue != spotitube.Version {
								ui.Append(fmt.Sprintf("Going to update from %d to %d version.", spotitube.Version, versionValue))
								subUpdateSoftware(versionResponseBody)
							}
							ui.Append(fmt.Sprintf("Actual version %d, online version %d.", spotitube.Version, versionValue), cui.DebugAppend)
						}
					}
				}
			}
		}
	}
}

func subFetchIndex() {
	if !system.FileExists(userLocalIndex) {
		ui.Append("No track index has been found.", cui.DebugAppend)
		return
	}
	ui.Append("Fetching local index...", cui.DebugAppend)
	system.FetchGob(userLocalIndex, tracksIndex)
}

func subWriteIndex() {
	ui.Append(fmt.Sprintf("Writing %d entries index...", len(tracksIndex.Tracks)), cui.DebugAppend)
	if writeErr := system.DumpGob(userLocalIndex, tracksIndex); writeErr != nil {
		ui.Append(fmt.Sprintf("Unable to write tracks index: %s", writeErr.Error()), cui.WarningAppend)
	}
}

func subAlignIndex() {
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		if linkPath, err := os.Readlink(path); err == nil {
			ui.Append(fmt.Sprintf("Index link: path %s", path), cui.DebugAppend)
			tracksIndex.Links[linkPath] = append(tracksIndex.Links[linkPath], path)
		} else if filepath.Ext(path) == ".mp3" {
			ui.Append(fmt.Sprintf("Index: path %s", path), cui.DebugAppend)
			spotifyID := track.GetTag(path, track.ID3FrameSpotifyID)
			if len(spotifyID) > 0 {
				tracksIndex.Tracks[spotifyID] = path
			} else {
				ui.Append(fmt.Sprintf("Index: no ID found. Ignoring %s...", path), cui.DebugAppend)
			}
		}

		return nil
	})
	waitIndex <- true
	ui.Append(fmt.Sprintf("Indexing finished: %d tracks indexed and %d linked.", len(tracksIndex.Tracks), len(tracksIndex.Links)))
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
		ui.Append(fmt.Sprintf("Unable to unmarshal Github latest relase data: %s", unmarshalErr.Error()), cui.WarningAppend)
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
		ui.Append(fmt.Sprintf("Unable to create temporary updated binary file: %s", err.Error()), cui.WarningAppend)
		return
	}
	defer binaryOutput.Close()

	ui.Append(fmt.Sprintf("Downloading update from %s...", binaryURL))
	binaryPayload, err := http.Get(binaryURL)
	if err != nil {
		ui.Append(fmt.Sprintf("Unable to download from %s: %s", binaryURL, err.Error()), cui.WarningAppend)
		return
	}
	defer binaryPayload.Body.Close()

	_, err = io.Copy(binaryOutput, binaryPayload.Body)
	if err != nil {
		ui.Append(fmt.Sprintf("Error while downloading from %s: %s", binaryURL, err.Error()), cui.WarningAppend)
		return
	}

	if user, err := user.Current(); err == nil {
		var (
			binaryFolder = fmt.Sprintf("%s/.spotitube", user.HomeDir)
			binaryFname  = fmt.Sprintf("%s/spotitube", binaryFolder)
		)
		err = system.Mkdir(binaryFolder)
		if err != nil {
			ui.Append(fmt.Sprintf("Unable to create binary container folder at %s: %s", binaryFolder, err.Error()), cui.WarningAppend)
			return
		}
		os.Remove(binaryFname)
		err = system.FileCopy(binaryTempFname, binaryFname)
		if err != nil {
			ui.Append(fmt.Sprintf("Unable to persist new binary to %s: %s", binaryFname, err.Error()), cui.WarningAppend)
			return
		}
		os.Remove(binaryTempFname)

		err = os.Chmod(binaryFname, 0755)
		if err != nil {
			ui.Append(fmt.Sprintf("Unable to make %s executable: %s", binaryFname, err.Error()), cui.WarningAppend)
			return
		}

		err = syscall.Exec(binaryFname, os.Args, os.Environ())
		if err != nil {
			ui.Append(fmt.Sprintf("Unable to exec updated instance: %s", err.Error()), cui.ErrorAppend)
		}
		mainExit()
	} else {
		return
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
		if !argFlushMissing && !argFlushDifferent {
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
		subCondFlushID3FrameOrigin(t, trackMp3)
		subCondFlushID3FrameDuration(t, trackMp3)
		subCondFlushID3FrameSpotifyID(t, trackMp3)
		subCondFlushID3FrameLyrics(t, trackMp3)
		trackMp3.Save()
	}
	trackMp3.Close()
}

func subCondFlushID3FrameTitle(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Title) > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameTitle))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameTitle) != t.Title)) {
		ui.Append("Inflating title metadata...", cui.DebugAppend)
		trackMp3.SetTitle(t.Title)
	}
}

func subCondFlushID3FrameSong(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Song) > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameSong))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameSong) != t.Song)) {
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
	if len(t.Artist) > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameArtist))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameArtist) != t.Artist)) {
		ui.Append("Inflating artist metadata...", cui.DebugAppend)
		trackMp3.SetArtist(t.Artist)
	}
}

func subCondFlushID3FrameAlbum(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Album) > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameAlbum))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameAlbum) != t.Album)) {
		ui.Append("Inflating album metadata...", cui.DebugAppend)
		trackMp3.SetAlbum(t.Album)
	}
}

func subCondFlushID3FrameGenre(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Genre) > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameGenre))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameGenre) != t.Genre)) {
		ui.Append("Inflating genre metadata...", cui.DebugAppend)
		trackMp3.SetGenre(t.Genre)
	}
}

func subCondFlushID3FrameYear(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Year) > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameYear))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameYear) != t.Year)) {
		ui.Append("Inflating year metadata...", cui.DebugAppend)
		trackMp3.SetYear(t.Year)
	}
}

func subCondFlushID3FrameFeaturings(t track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Featurings) > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameFeaturings))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameFeaturings) != strings.Join(t.Featurings, "|"))) {
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
	if t.TrackNumber > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameTrackNumber))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameTrackNumber) != fmt.Sprintf("%d", t.TrackNumber))) {
		ui.Append("Inflating track number metadata...", cui.DebugAppend)
		trackMp3.AddFrame(trackMp3.CommonID("Track number/Position in set"),
			id3v2.TextFrame{
				Encoding: id3v2.EncodingUTF8,
				Text:     strconv.Itoa(t.TrackNumber),
			})
	}
}

func subCondFlushID3FrameTrackTotals(t track.Track, trackMp3 *id3v2.Tag) {
	if t.TrackTotals > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameTrackTotals))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameTrackTotals) != fmt.Sprintf("%d", t.TrackTotals))) {
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
	if system.FileExists(t.FilenameArtwork()) &&
		(!argFlushMissing || (argFlushMissing && !t.HasID3Frame(track.ID3FrameArtwork))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameArtworkURL) != t.Image)) {
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
	if len(t.Image) > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameArtworkURL))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameArtworkURL) != t.Image)) {
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
	if len(t.URL) > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameOrigin))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameOrigin) != t.URL)) {
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
	if t.Duration > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameDuration))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameDuration) != fmt.Sprintf("%d", t.Duration))) {
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
	if len(t.SpotifyID) > 0 &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameSpotifyID))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameSpotifyID) != t.SpotifyID)) {
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
	if len(t.Lyrics) > 0 && !argDisableLyrics &&
		(!argFlushMissing || (argFlushMissing && !track.TagHasFrame(trackMp3, track.ID3FrameLyrics))) &&
		(!argFlushDifferent || (argFlushDifferent && track.TagGetFrame(trackMp3, track.ID3FrameLyrics) != t.Lyrics)) {
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
	return !t.Local || argReplaceLocal || argSimulate
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
	if argReplaceLocal {
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
	return !t.Local || argFlushMetadata || argReplaceLocal
}

func subCondSequentialDo(t *track.Track) {
	if (t.Local && argFlushMetadata) || !t.Local {
		subCondLyricsFetch(t)
		subCondArtworkDownload(t)
	}
}

func subCondLyricsFetch(t *track.Track) {
	if !argDisableLyrics &&
		(!argFlushMissing || (argFlushMissing && !t.HasID3Frame(track.ID3FrameLyrics))) {
		ui.Append(fmt.Sprintf("Fetching song \"%s\" lyrics...", t.Basename()), cui.DebugAppend)
		lyricsErr := t.SearchLyrics()
		if lyricsErr != nil {
			ui.Append(fmt.Sprintf("Something went wrong while searching for song lyrics: %s", lyricsErr.Error()), cui.WarningAppend)
		} else {
			ui.Append(fmt.Sprintf("Song lyrics found."), cui.DebugAppend)
		}
	}
}

func subCondArtworkDownload(t *track.Track) {
	if len(t.Image) > 0 && !system.FileExists(t.FilenameArtwork()) &&
		(!argFlushMissing || (argFlushMissing && !t.HasID3Frame(track.ID3FrameArtwork))) {
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
