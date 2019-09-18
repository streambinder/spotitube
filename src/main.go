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
	// flags for separate flows
	argCleanJunks bool
	argVersion    bool
	// flags for media sources
	argLibrary   bool
	argAlbums    system.StringsFlag
	argPlaylists system.StringsFlag
	argTracksFix system.StringsFlag
	// flags for options
	argFolder                string
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
	// flags for troubleshooting
	argLog        bool
	argDebug      bool
	argSimulate   bool
	argDisableGui bool

	// spotify
	c         *spotify.Client
	cUser     string
	cUserID   string
	tracks    []*track.Track
	playlists []*track.Playlist
	index     *track.TracksIndex

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
	mainAuthenticate()
	mainFetch()
	mainSearch()
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
	// separate flows
	flag.BoolVar(&argCleanJunks, "clean-junks", false, "Scan for and clean junk files")
	flag.BoolVar(&argVersion, "version", false, "Print version")

	// media sources
	flag.BoolVar(&argLibrary, "library", false, "Synchronize user library")
	flag.Var(&argAlbums, "album", "Album URI to synchronize")
	flag.Var(&argPlaylists, "playlist", "Playlist URI to synchronize")
	flag.Var(&argTracksFix, "fix", "Offline song filename(s) which straighten the shot to")

	// options
	flag.StringVar(&argFolder, "folder", ".", "Folder to sync your tracks collection into")
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

	// troubleshooting
	flag.BoolVar(&argLog, "log", false, "Enable logging into file ./spotitube.log")
	flag.BoolVar(&argDebug, "debug", false, "Enable debug messages")
	flag.BoolVar(&argSimulate, "simulate", false, "Simulate process flow, without really altering filesystem")
	// TODO: gocui should work on windows, too
	if runtime.GOOS != "windows" {
		flag.BoolVar(&argDisableGui, "disable-gui", false, "Disable GUI to reduce noise and increase readability of program flow")
	} else {
		argDisableGui = true
	}
	flag.Parse()

	if !(argAlbums.IsSet() || argPlaylists.IsSet() || argTracksFix.IsSet()) {
		argLibrary = true
	}

	// TODO: fix this case
	if len(argTracksFix.Entries) > 0 {
		argFlushLocal = true
		argFlushMetadata = true
	}

	if argManualInput {
		argInteractive = true
	}

	if argAuthenticateOutside {
		argDisableBrowserOpening = true
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
			system.FetchGob(spotitube.UserIndex, index)
		}
		index = track.Index(argFolder)
	}

	for range [spotitube.ConcurrencyLimit]int{} {
		waitGroupPool <- true
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

func mainAuthenticate() {
	host := "localhost"
	if argAuthenticateOutside {
		host = "spotitube.local"
		ui.Prompt("Outside authentication enabled: assure \"spotitube.local\" points to this machine.")
	}

	uri := spotify.BuildAuthURL(host)
	ui.Append(fmt.Sprintf("Authentication URL: %s", uri.Short), cui.ParagraphAutoReturn)
	if !argDisableBrowserOpening {
		ui.Append("Waiting for automatic login process. If wait is too long, manually open that URL.", cui.DebugAppend)
	}

	var err error
	if c, err = spotify.Auth(uri.Full, host, !argDisableBrowserOpening); err != nil {
		ui.Prompt(fmt.Sprintf("Authentication failed: %s", err.Error()), cui.PromptExit)
	}

	cUser, cUserID = c.User()
	ui.Append("Authentication completed.")
	ui.Append(fmt.Sprintf("%s %s", cui.Font("Session user:", cui.StyleBold), cUser), cui.PanelLeftTop)
}

func mainFetch() {

	mainFetchLibrary()
	mainFetchAlbums()
	mainFetchPlaylists()
	mainFetchTracksToFix()

	// ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs duplicates:", cui.StyleBold), len(tracksDuplicates)), cui.PanelLeftTop)
	ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs online:", cui.StyleBold), len(tracks)), cui.PanelLeftTop)
	ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs offline:", cui.StyleBold), track.CountOffline(tracks)), cui.PanelLeftTop)
	ui.Append(fmt.Sprintf("%s %d", cui.Font("Songs missing:", cui.StyleBold), track.CountOnline(tracks)), cui.PanelLeftTop)

	track.IndexWait()

	if len(tracks) == 0 {
		ui.Prompt("No song needs to be downloaded.", cui.PromptExit)
		mainExit()
	}

}

func mainFetchLibrary() {
	if !argLibrary {
		return
	}

	gob := fmt.Sprintf(spotitube.UserGob, cUserID, "library")
	if argFlushCache {
		os.Remove(gob)
	}

	dump, dumpErr := subFetchGob(gob)
	if dumpErr == nil && time.Since(dump.Time) < spotitube.TracksCacheDuration {
		ui.Append(fmt.Sprintf("%s %d/%d (min)", cui.Font("Library cache expiration:", cui.StyleBold), int(time.Since(dump.Time)), spotitube.TracksCacheDuration), cui.PanelLeftTop)
		for _, t := range dump.Tracks {
			tracks = append(tracks, t)
		}
		return
	}

	ui.Append("Fetching music library...")
	library, err := c.LibraryTracks()
	if err != nil {
		ui.Prompt(fmt.Sprintf("Unable to fetch library: %s", err.Error()), cui.PromptExit)
	}

	if err := system.DumpGob(gob, track.TracksDump{Tracks: library, Time: time.Now()}); err != nil {
		ui.Append(fmt.Sprintf("Unable to cache tracks: %s", err.Error()), cui.WarningAppend)
	}

	dup := make(map[spotify.ID]float64)
	for _, t := range library {
		if _, isDup := dup[spotify.ID(t.SpotifyID)]; !isDup {
			tracks = append(tracks, t)
		}
		dup[spotify.ID(t.SpotifyID)] = 1
	}

	if argRemoveDuplicates {
		ids := []spotify.ID{}
		for id := range dup {
			ids = append(ids, id)
		}

		err := c.RemoveLibraryTracks(ids)
		if err != nil {
			ui.Append(fmt.Sprintf("Unable to remove library duplicates: %s", err.Error()), cui.WarningAppend)
		}
	}
}

func mainFetchAlbums() {
	if !argAlbums.IsSet() {
		return
	}

	for _, uri := range argAlbums.Entries {
		gob := fmt.Sprintf(spotitube.UserGob, cUserID, fmt.Sprintf("album_%s", spotify.IDFromURI(uri)))
		if argFlushCache {
			os.Remove(gob)
		}

		dump, dumpErr := subFetchGob(gob)
		if dumpErr == nil && time.Since(dump.Time) < spotitube.TracksCacheDuration {
			ui.Append(fmt.Sprintf("%s %d/%d (min)", cui.Font(fmt.Sprintf("Album %s cache expiration:", spotify.IDFromURI(uri)), cui.StyleBold), int(time.Since(dump.Time)), spotitube.TracksCacheDuration), cui.PanelLeftTop)
			for _, t := range dump.Tracks {
				tracks = append(tracks, t)
			}
			continue
		}

		ui.Append(fmt.Sprintf("Fetching album %s...", spotify.IDFromURI(uri)))
		album, err := c.AlbumTracks(uri)
		if err != nil {
			ui.Prompt(fmt.Sprintf("Unable to fetch album: %s", err.Error()), cui.PromptExit)
		}

		if err := system.DumpGob(gob, track.TracksDump{Tracks: album, Time: time.Now()}); err != nil {
			ui.Append(fmt.Sprintf("Unable to cache tracks: %s", err.Error()), cui.WarningAppend)
		}

		for _, t := range album {
			tracks = append(tracks, t)
		}
	}
}

func mainFetchPlaylists() {
	if !argPlaylists.IsSet() {
		return
	}

	for _, uri := range argPlaylists.Entries {
		playlist := &track.Playlist{}
		if p, err := c.Playlist(uri); err == nil {
			playlist.Name = p.Name
			playlist.Owner = p.Owner.DisplayName
		}

		gob := fmt.Sprintf(spotitube.UserGob, cUserID, fmt.Sprintf("playlist_%s", spotify.IDFromURI(uri)))
		if argFlushCache {
			os.Remove(gob)
		}

		dump, dumpErr := subFetchGob(gob)
		if dumpErr == nil && time.Since(dump.Time) < spotitube.TracksCacheDuration {
			ui.Append(fmt.Sprintf("%s %d/%d (min)", cui.Font(fmt.Sprintf("Playlist %s cache expiration:", playlist.Name), cui.StyleBold), int(time.Since(dump.Time)), spotitube.TracksCacheDuration), cui.PanelLeftTop)
			for _, t := range dump.Tracks {
				tracks = append(tracks, t)
			}
			playlist.Tracks = dump.Tracks
			playlists = append(playlists, playlist)
			continue
		}

		ui.Append(fmt.Sprintf("Fetching playlist %s...", spotify.IDFromURI(uri)))
		p, err := c.PlaylistTracks(uri)
		if err != nil {
			ui.Prompt(fmt.Sprintf("Unable to fetch playlist: %s", err.Error()), cui.PromptExit)
		}

		if err := system.DumpGob(gob, track.TracksDump{Tracks: p, Time: time.Now()}); err != nil {
			ui.Append(fmt.Sprintf("Unable to cache tracks: %s", err.Error()), cui.WarningAppend)
		}

		dup := make(map[spotify.ID]float64)
		for _, t := range p {
			if _, isDup := dup[spotify.ID(t.SpotifyID)]; !isDup {
				tracks = append(tracks, t)
			}
			dup[spotify.ID(t.SpotifyID)] = 1
		}

		playlist.Tracks = dump.Tracks
		playlists = append(playlists, playlist)
		if argRemoveDuplicates {
			ids := []spotify.ID{}
			for id := range dup {
				ids = append(ids, id)
			}

			err := c.RemovePlaylistTracks(uri, ids)
			if err != nil {
				ui.Append(fmt.Sprintf("Unable to remove %s playlist duplicates: %s", uri, err.Error()), cui.WarningAppend)
			}
		}
	}
}

func mainFetchTracksToFix() {
	if !argTracksFix.IsSet() {
		return
	}

	ui.Append(fmt.Sprintf("%s %d", cui.Font("Fix song(s):", cui.StyleBold), len(argTracksFix.Entries)), cui.PanelLeftTop)
	for _, tFix := range argTracksFix.Entries {
		if t, err := track.OpenLocalTrack(tFix); err != nil {
			ui.Prompt(fmt.Sprintf("Something went wrong: %s", err.Error()))
		} else {
			tracks = append(tracks, t)
		}
	}
}

func mainSearch() {
	songsFetch, songsFlush, songsIgnore := subCountSongs()

	ui.ProgressMax = len(tracks)
	ui.Append(fmt.Sprintf("%d will be downloaded, %d flushed and %d ignored", songsFetch, songsFlush, songsIgnore))

	for i, t := range tracks {
		ui.ProgressIncrease()
		ui.Append(fmt.Sprintf("%d/%d: \"%s\"", i+1, len(tracks), t.Basename()), cui.StyleBold)

		if path, ok := index.Tracks[t.SpotifyID]; ok {
			if !strings.Contains(path, fmt.Sprintf("%c%s", os.PathSeparator, t.Filename())) {
				ui.Append(fmt.Sprintf("Track %s vs %s has been renamed: moving local one to %s", path, t.Filename(), t.Filename()))
				if err := subTrackRename(t); err != nil {
					ui.Append(fmt.Sprintf("Unable to rename: %s", err.Error()), cui.ErrorAppend)
				}
			}
		}

		if !t.Local() || argFlushLocal || argSimulate {
			var (
				entry    = new(provider.Entry)
				prov     provider.Provider
				provName string
			)

			for provName, prov = range provider.Providers {
				ui.Append(fmt.Sprintf("Searching entries on %s provider", provName))

				var (
					provEntries   = []*provider.Entry{}
					provErr       error
					entryPickAuto bool
					entryPick     bool
				)

				if !argManualInput {
					provEntries, provErr = prov.Query(t)
					if provErr != nil {
						ui.Append(fmt.Sprintf("Something went wrong while searching for \"%s\" track: %s.", t.Basename(), provErr.Error()), cui.WarningAppend)
						// FIXME: tracksFailed = append(tracksFailed, t)
						continue
					}

					for _, provEntry := range provEntries {
						ui.Append(fmt.Sprintf("Result met: ID: %s,\nTitle: %s,\nUser: %s,\nDuration: %d.",
							provEntry.ID, provEntry.Title, provEntry.User, provEntry.Duration), cui.DebugAppend)

						entryPickAuto, entryPick = subMatchResult(prov, t, provEntry)
						if subIfPickFromAns(entryPickAuto, entryPick) {
							ui.Append(fmt.Sprintf("Video \"%s\" is good to go for \"%s\".", provEntry.Title, t.Basename()))
							entry = provEntry
							break
						}
					}
				}

				subCondManualInputURL(prov, entry, t)
				if entry.URL == "" {
					entry = &provider.Entry{}
				}

				if entry == (&provider.Entry{}) {
					ui.Append(fmt.Sprintf("Video for \"%s\" not found.", t.Basename()), cui.ErrorAppend)
					// FIXME: tracksFailed = append(tracksFailed, t)
					continue
				}

				if argSimulate {
					ui.Append(fmt.Sprintf("I would like to download \"%s\" for \"%s\" track, but I'm just simulating.", entry.URL, t.Basename()))
					continue
				} else if argFlushLocal {
					if t.URL == entry.URL && !entryPick {
						ui.Append(fmt.Sprintf("Track \"%s\" is still the best result I can find.", t.Basename()))
						ui.Append(fmt.Sprintf("Local track origin URL %s is the same as the chosen one %s.", t.URL, entry.URL), cui.DebugAppend)
						continue
					} else {
						t.URL = ""
					}
				}
			}

			ui.Append(fmt.Sprintf("Going to download \"%s\" from %s...", entry.Title, entry.URL))
			err := prov.Download(entry, t.FilenameTemporary())
			if err != nil {
				ui.Append(fmt.Sprintf("Something went wrong downloading \"%s\": %s.", t.Basename(), err.Error()), cui.WarningAppend)
				// FIXME: tracksFailed = append(tracksFailed, t)
				continue
			} else {
				t.URL = entry.URL
			}
		}

		if !subIfSongProcess(*t) {
			continue
		}

		subCondSequentialDo(t)

		ui.Append(fmt.Sprintf("Launching song processing jobs..."))
		waitGroup.Add(1)
		go subParallelSongProcess(*t, &waitGroup)
		if argDebug {
			waitGroup.Wait()
		}
	}
	waitGroup.Wait()

	subCondPlaylistFileWrite()
	subCondTimestampFlush()

	index.Sync(spotitube.UserIndex)

	close(waitGroupPool)
	waitGroup.Wait()
	ui.ProgressFill()

	// FIXME
	// ui.Append(fmt.Sprintf("%d tracks failed to synchronize.", len(tracksFailed)))
	// for _, t := range tracksFailed {
	// 	ui.Append(fmt.Sprintf(" - \"%s\"", t.Basename()))
	// }
	//
	// system.Notify("SpotiTube", "emblem-downloads", "SpotiTube", fmt.Sprintf("%d track(s) synced, %d failed.", len(tracks)-len(tracksFailed), len(tracksFailed)))

	system.Notify("SpotiTube", "emblem-downloads", "SpotiTube", fmt.Sprintf("%d track(s) synced", len(tracks)))
	ui.Prompt("Synchronization completed.", cui.PromptExit)
}

func mainExit(delay ...time.Duration) {
	system.FileWildcardDelete(argFolder, track.JunkWildcards()...)

	if len(delay) > 0 {
		time.Sleep(delay[0])
	}

	os.Exit(0)
}

func subParallelSongProcess(t track.Track, wg *sync.WaitGroup) {
	defer ui.ProgressHalfIncrease()
	defer wg.Done()
	<-waitGroupPool

	if !t.Local() && !argDisableNormalization {
		subSongNormalize(t)
	}

	if !system.FileExists(t.FilenameTemporary()) && system.FileExists(t.Filename()) {
		if err := system.FileCopy(t.Filename(), t.FilenameTemporary()); err != nil {
			ui.Append(fmt.Sprintf("Unable to prepare song for getting its metadata flushed: %s", err.Error()), cui.WarningAppend)
			return
		}
	}

	if (t.Local() && argFlushMetadata) || !t.Local() {
		subSongFlushMetadata(t)
	}

	os.Remove(t.Filename())
	err := os.Rename(t.FilenameTemporary(), t.Filename())
	if err != nil {
		ui.Append(fmt.Sprintf("Unable to move song to its final path: %s", err.Error()), cui.WarningAppend)
	}

	waitGroupPool <- true
}

// TODO: move to command package
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

func subFetchGob(path string) (track.TracksDump, error) {
	var tracksDump = new(track.TracksDump)
	if fetchErr := system.FetchGob(path, tracksDump); fetchErr != nil {
		return track.TracksDump{}, fmt.Errorf(fmt.Sprintf("Unable to load tracks cache: %s", fetchErr.Error()))
	}

	if time.Since(tracksDump.Time) > spotitube.TracksCacheDuration {
		return track.TracksDump{}, fmt.Errorf("Tracks cache declared obsolete: flushing it from Spotify")
	}

	return *tracksDump, nil
}

func subCountSongs() (int, int, int) {
	var (
		fetch  int
		flush  int
		ignore int
	)

	if argFlushLocal {
		fetch = len(tracks)
		flush = fetch
	} else if argFlushMetadata {
		fetch = track.CountOnline(tracks)
		flush = len(tracks)
	} else {
		fetch = track.CountOnline(tracks)
		flush = fetch
		ignore = track.CountOffline(tracks)
	}

	return fetch, flush, ignore
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
	return !t.Local() || argFlushMetadata || argFlushLocal
}

func subCondSequentialDo(t *track.Track) {
	if (t.Local() && argFlushMetadata) || !t.Local() {
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
	if argSimulate || argDisablePlaylistFile || len(argPlaylists.Entries) == 0 {
		return
	}

	for _, p := range playlists {
		var (
			pFolder  = slug.Make(p.Name)
			pFname   = fmt.Sprintf("%s/%s", pFolder, p.Name)
			pContent string
		)

		pFname = pFname + ".m3u"
		if argPlsFile {
			pFname = pFname + ".pls"
		}

		os.RemoveAll(pFolder)
		os.Mkdir(pFolder, 0775)
		os.Chdir(pFolder)
		for _, t := range tracks {
			if system.FileExists("../" + t.Filename()) {
				if err := os.Symlink("../"+t.Filename(), t.Filename()); err != nil {
					ui.Append(fmt.Sprintf("Unable to create symlink for \"%s\" in %s: %s", t.Filename(), pFolder, err.Error()), cui.ErrorAppend)
				}
			}
		}
		os.Chdir("..")

		ui.Append(fmt.Sprintf("Creating playlist file at %s...", pFname))
		if system.FileExists(pFname) {
			os.Remove(pFname)
		}

		if argPlsFile {
			pContent = p.PLS()
		} else {
			pContent = p.M3U()
		}

		pFile, err := os.Create(pFname)
		if err != nil {
			ui.Append(fmt.Sprintf("Unable to create playlist file: %s", err.Error()), cui.WarningAppend)
			continue
		}
		defer pFile.Close()
		defer pFile.Sync()

		if _, err := pFile.WriteString(pContent); err != nil {
			ui.Append(fmt.Sprintf("Unable to write playlist file: %s", err.Error()), cui.WarningAppend)
		}
	}
}

func subTrackRename(t *track.Track) error {
	var (
		keyID   = t.SpotifyID
		keyPath = index.Tracks[t.SpotifyID]
	)
	if err := os.Rename(keyPath, t.Filename()); err != nil {
		return err
	}

	for _, trackLink := range index.Links[keyPath] {
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
		index.Links[t.Filename()] = append(index.Links[t.Filename()], trackLinkNew)
	}
	delete(index.Links, keyPath)
	index.Tracks[keyID] = t.Filename()

	return nil
}
