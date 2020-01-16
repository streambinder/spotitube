package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bogem/id3v2"
	"github.com/gosimple/slug"
	"github.com/streambinder/spotitube/cui"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/provider"
	"github.com/streambinder/spotitube/shell"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/system"
	"github.com/streambinder/spotitube/track"
	"github.com/streambinder/spotitube/upstream"
)

const (
	version          = 25
	cacheDuration    = 30 * time.Minute
	concurrencyLimit = 100
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
	tracks    = make(map[*track.Track]*track.SyncOptions)
	playlists []*track.Playlist
	index     *track.TracksIndex

	// routines
	waitGroup     sync.WaitGroup
	waitGroupPool = make(chan bool, concurrencyLimit)

	// cli
	ui *cui.CUI

	// user paths
	usrBinary = fmt.Sprintf("%s/spotitube", usrPath())
	usrIndex  = fmt.Sprintf("%s/index.gob", usrPath())
	usrGob    = fmt.Sprintf("%s/%s_%s.gob", usrPath(), "%s", "%s")
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
	system.Mkdir(usrPath())
}

func mainFork() {
	if system.Proc() != usrBinary &&
		system.FileExists(usrBinary) &&
		os.Getenv("SPOTITUBE_FORKED") != "1" {
		syscall.Exec(usrBinary, os.Args, append(os.Environ(), []string{"SPOTITUBE_FORKED=1"}...))
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

	if !shell.YoutubeDL().Exists() {
		fmt.Println(fmt.Sprintf("%s command is not installed.", shell.YoutubeDL().Name()))
		os.Exit(1)
	}

	if !shell.FFmpeg().Exists() {
		fmt.Println(fmt.Sprintf("%s command is not installed.", shell.FFmpeg().Name()))
		os.Exit(1)
	}

	if !system.IsOnline() {
		fmt.Println("You're not connected to the internet.")
		os.Exit(1)
	}

	if upstreamVersion, err := upstream.Version(); err == nil &&
		!argDisableUpdateCheck && upstreamVersion != version {
		fmt.Println(fmt.Sprintf("You're not running latest version (%d). Going to update.", upstreamVersion))
		if err := upstream.Download(usrBinary); err != nil {
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

	if argAuthenticateOutside {
		argDisableBrowserOpening = true
	}
}

func mainInit() {
	system.TrapSignal(os.Interrupt, func() {
		mainExit()
	})

	if argVersion {
		fmt.Println(fmt.Sprintf("SpotiTube, version %d.", version))
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
		if system.FileExists(usrIndex) {
			system.FetchGob(usrIndex, index)
		}
		index = track.Index(argFolder)
	}

	for range [concurrencyLimit]int{} {
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
	ui.Append(fmt.Sprintf("%s %s", cui.Font(fmt.Sprintf("%s version:", shell.YoutubeDL().Name()), cui.StyleBold), shell.YoutubeDL().Version()), cui.PanelLeftTop)
	ui.Append(fmt.Sprintf("%s %s", cui.Font(fmt.Sprintf("%s version:", shell.FFmpeg().Name()), cui.StyleBold), shell.FFmpeg().Version()), cui.PanelLeftTop)
	ui.Append(fmt.Sprintf("%s %d", cui.Font("Version:", cui.StyleBold), version), cui.PanelLeftBottom)
	ui.Append(fmt.Sprintf("%s %s", cui.Font("Proc:", cui.StyleBold), system.PrettyPath(system.Proc())), cui.PanelLeftBottom)
	ui.Append(fmt.Sprintf("%s %s", cui.Font("Date:", cui.StyleBold), time.Now().Format("2006-01-02 15:04:05")), cui.PanelLeftBottom)
	ui.Append(fmt.Sprintf("%s GPL-3.0", cui.Font("License:", cui.StyleBold)), cui.PanelLeftBottom)
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

	if argFlushLocal || argFlushMetadata {
		for _, opts := range tracks {
			if argFlushLocal {
				opts.Source = true
			}
			if argFlushMetadata {
				opts.Metadata = true
			}
		}
	}

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

	gob := fmt.Sprintf(usrGob, cUserID, "library")
	if argFlushCache {
		os.Remove(gob)
	}

	dump, dumpErr := subFetchGob(gob)
	if dumpErr == nil && time.Since(dump.Time) < cacheDuration {
		ui.Append(fmt.Sprintf("%s %d/%d (min)", cui.Font("Library cache expiration:", cui.StyleBold), int(time.Since(dump.Time)), cacheDuration), cui.PanelLeftTop)
		for _, t := range dump.Tracks {
			tracks[t] = track.SyncOptionsDefault()
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
			tracks[t] = track.SyncOptionsDefault()
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
		gob := fmt.Sprintf(usrGob, cUserID, fmt.Sprintf("album_%s", spotify.IDFromURI(uri)))
		if argFlushCache {
			os.Remove(gob)
		}

		dump, dumpErr := subFetchGob(gob)
		if dumpErr == nil && time.Since(dump.Time) < cacheDuration {
			ui.Append(fmt.Sprintf("%s %d/%d (min)", cui.Font(fmt.Sprintf("Album %s cache expiration:", spotify.IDFromURI(uri)), cui.StyleBold), int(time.Since(dump.Time)), cacheDuration), cui.PanelLeftTop)
			for _, t := range dump.Tracks {
				tracks[t] = track.SyncOptionsDefault()
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
			tracks[t] = track.SyncOptionsDefault()
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

		gob := fmt.Sprintf(usrGob, cUserID, fmt.Sprintf("playlist_%s", spotify.IDFromURI(uri)))
		if argFlushCache {
			os.Remove(gob)
		}

		dump, dumpErr := subFetchGob(gob)
		if dumpErr == nil && time.Since(dump.Time) < cacheDuration {
			ui.Append(fmt.Sprintf("%s %d/%d (min)", cui.Font(fmt.Sprintf("Playlist %s cache expiration:", playlist.Name), cui.StyleBold), int(time.Since(dump.Time)), cacheDuration), cui.PanelLeftTop)
			for _, t := range dump.Tracks {
				tracks[t] = track.SyncOptionsDefault()
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
				tracks[t] = track.SyncOptionsDefault()
			}
			dup[spotify.ID(t.SpotifyID)] = 1
		}

		playlist.Tracks = p
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
			tracks[t] = track.SyncOptionsFlush()
		}
	}
}

func mainSearch() {
	ctr := 0
	songsFetch, songsFlush, songsIgnore := subCountSongs()

	ui.ProgressMax = len(tracks)
	ui.Append(fmt.Sprintf("%d will be downloaded, %d flushed and %d ignored", songsFetch, songsFlush, songsIgnore))

	for track, trackOpts := range tracks {
		ctr++
		ui.ProgressIncrease()
		ui.Append(fmt.Sprintf("%d/%d: \"%s\"", ctr, len(tracks), track.Basename()), cui.StyleBold)

		// rename local file if Spotify has renamed it
		if path, match, err := index.Match(track.SpotifyID, track.Filename()); err == nil && !match {
			ui.Append(fmt.Sprintf("Track %s has been renamed: moving to %s", track.Filename(), path))
			if err := os.Rename(path, track.Filename()); err != nil {
				ui.Append(fmt.Sprintf("Unable to rename: %s", err.Error()), cui.ErrorAppend)
			} else {
				index.Rename(track.SpotifyID, track.Filename())
			}
		}

		if !track.Local() || trackOpts.Source || argSimulate {
			entry := new(provider.Entry)

			for _, p := range provider.All() {
				ui.Append(fmt.Sprintf("Searching entries on %s provider", p.Name()))

				var (
					provEntries = []*provider.Entry{}
					provErr     error
					entryPick   bool
				)

				if !argManualInput {
					provEntries, provErr = p.Query(track)
					if provErr != nil {
						ui.Append(
							fmt.Sprintf("Unable to search %s on %s provider: %s.", track.Basename(), p.Name(), provErr.Error()),
							cui.WarningAppend)
						// FIXME: tracksFailed = append(tracksFailed, t)
						continue
					}

					for _, provEntry := range provEntries {
						ui.Append(
							fmt.Sprintf("Result met: ID: %s,\nTitle: %s,\nUser: %s,\nDuration: %d.",
								provEntry.ID, provEntry.Title, provEntry.User, provEntry.Duration),
							cui.DebugAppend)

						entryPick = bool(p.Match(provEntry, track) == nil)
						if argInteractive {
							entryPick = ui.Prompt(
								fmt.Sprintf(
									"Track: %s\n\nID: %s\nTitle: %s\nUser: %s\nDuration: %d\nURL: %s\nResult is matching: %s",
									track.Basename(), entry.ID, entry.Title, entry.User,
									entry.Duration, entry.URL, strconv.FormatBool(entryPick)),
								cui.PromptBinary)
						}

						if entryPick {
							ui.Append(fmt.Sprintf("Video \"%s\" is good to go for \"%s\".", provEntry.Title, track.Basename()))
							entry = provEntry
							break
						}
					}
				}

				if argManualInput && entry.Empty() {
					if url := ui.PromptInputMessage(fmt.Sprintf("Enter URL for \"%s\"", track.Basename()), cui.PromptInput); len(url) > 0 {
						if err := p.Support(url); err == nil {
							entry.URL = url
						} else {
							ui.Prompt(fmt.Sprintf("Something went wrong: %s", err.Error()))
						}
					}
				}

				if entry.Empty() {
					ui.Append("No entry to download has been found.", cui.ErrorAppend)
					// FIXME: tracksFailed = append(tracksFailed, t)
					continue
				}

				if argSimulate {
					ui.Append(fmt.Sprintf("I would like to download \"%s\" for \"%s\" track, but I'm just simulating.", entry.Repr(), track.Basename()))
					continue
				}

				if trackOpts.Source && track.URL == entry.URL {
					ui.Append("Downloaded track is still the best result I can find.")
					ui.Append(fmt.Sprintf("Local track origin URL %s is the same as the chosen one %s.", track.URL, entry.URL), cui.DebugAppend)
					continue
				}
			}

			if entry.Empty() {
				continue
			}

			ui.Append(fmt.Sprintf("Going to download \"%s\" from %s...", entry.Title, entry.Repr()))
			p, err := provider.For(entry.URL)
			if err != nil {
				ui.Append(fmt.Sprintf("Unable to reconstruct provider for \"%s\"", entry.URL), cui.ErrorAppend)
				continue
			}

			if err := p.Download(entry, track.FilenameTemporary()); err != nil {
				ui.Append(fmt.Sprintf("Something went wrong downloading \"%s\": %s.", track.Basename(), err.Error()), cui.WarningAppend)
				// FIXME: tracksFailed = append(tracksFailed, t)
				continue
			}

			track.URL = entry.URL
		}

		if track.Local() && !trackOpts.Metadata {
			continue
		}

		mainSearchLyrics(track)
		mainDownloadArtwork(track)

		ui.Append(fmt.Sprintf("Launching track processing jobs..."))
		waitGroup.Add(1)
		go subParallelSongProcess(track, trackOpts, &waitGroup)
		if argDebug {
			waitGroup.Wait()
		}
	}

	waitGroup.Wait()

	subCondPlaylistFileWrite()
	subCondTimestampFlush()

	index.Sync(usrIndex)

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

func mainSearchLyrics(t *track.Track) {
	if argDisableLyrics {
		return
	}

	for _, p := range lyrics.All() {
		ui.Append(fmt.Sprintf("Searching lyrics using %s provider...", p.Name()))

		text, err := p.Query(t.Song, t.Artist)
		if err != nil {
			ui.Append(fmt.Sprintf("Provider encountered an error: %s", err.Error()))
			continue
		}

		t.Lyrics = text
		break
	}
}

func mainDownloadArtwork(t *track.Track) {
	if len(t.Image) == 0 || system.FileExists(t.FilenameArtwork()) {
		return
	}

	ui.Append(fmt.Sprintf("Downloading track artwork %s...", t.Image), cui.DebugAppend)
	if err := system.Wget(t.Image, t.FilenameArtwork()); err != nil {
		ui.Append(fmt.Sprintf("Unable to download artwork: %s", err.Error()), cui.WarningAppend)
	}
}

func mainExit(delay ...time.Duration) {
	system.FileWildcardDelete(argFolder, track.JunkWildcards()...)

	if len(delay) > 0 {
		time.Sleep(delay[0])
	}

	os.Exit(0)
}

func subParallelSongProcess(track *track.Track, trackOpts *track.SyncOptions, wg *sync.WaitGroup) {
	defer wg.Done()
	<-waitGroupPool

	if !track.Local() && !argDisableNormalization {
		subSongNormalize(track)
	}

	if !system.FileExists(track.FilenameTemporary()) && system.FileExists(track.Filename()) {
		if err := system.FileCopy(track.Filename(), track.FilenameTemporary()); err != nil {
			ui.Append(fmt.Sprintf("Unable to prepare song for getting its metadata flushed: %s", err.Error()), cui.WarningAppend)
			return
		}
	}

	if !track.Local() || trackOpts.Metadata {
		subSongFlushMetadata(track)
	}

	os.Remove(track.Filename())
	err := os.Rename(track.FilenameTemporary(), track.Filename())
	if err != nil {
		ui.Append(fmt.Sprintf("Unable to move song to its final path: %s", err.Error()), cui.WarningAppend)
	}

	waitGroupPool <- true
}

func subSongNormalize(t *track.Track) {
	volume, err := shell.FFmpeg().VolumeDetect(t.FilenameTemporary())
	if err != nil {
		ui.Append(fmt.Sprintf("Unable to detect max volume value for %s: %s", t.Basename(), err.Error()), cui.WarningAppend)
		return
	}

	if volume > 0 {
		return
	}

	volume = math.Abs(volume)
	if err := shell.FFmpeg().VolumeSet(volume, t.FilenameTemporary()); err != nil {
		ui.Append(fmt.Sprintf("Unable to increase max volume by %f delta for %s: %s", volume, t.Basename(), err.Error()), cui.WarningAppend)
	}
}

func subSongFlushMetadata(t *track.Track) {
	trackMp3, err := id3v2.Open(t.FilenameTemporary(), id3v2.Options{Parse: true})
	defer trackMp3.Close()
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
}

func subCondFlushID3FrameTitle(t *track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Title) > 0 && track.TagGetFrame(trackMp3, track.ID3FrameSong) != track.TagGetFrame(trackMp3, track.ID3FrameTitle) {
		ui.Append("Inflating title metadata...", cui.DebugAppend)
		trackMp3.SetTitle(t.Title)
	}
}

func subCondFlushID3FrameSong(t *track.Track, trackMp3 *id3v2.Tag) {
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

func subCondFlushID3FrameArtist(t *track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Artist) > 0 && t.Artist != track.TagGetFrame(trackMp3, track.ID3FrameArtist) {
		ui.Append("Inflating artist metadata...", cui.DebugAppend)
		trackMp3.SetArtist(t.Artist)
	}
}

func subCondFlushID3FrameAlbum(t *track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Album) > 0 && t.Album != track.TagGetFrame(trackMp3, track.ID3FrameAlbum) {
		ui.Append("Inflating album metadata...", cui.DebugAppend)
		trackMp3.SetAlbum(t.Album)
	}
}

func subCondFlushID3FrameGenre(t *track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Genre) > 0 && t.Genre != track.TagGetFrame(trackMp3, track.ID3FrameGenre) {
		ui.Append("Inflating genre metadata...", cui.DebugAppend)
		trackMp3.SetGenre(t.Genre)
	}
}

func subCondFlushID3FrameYear(t *track.Track, trackMp3 *id3v2.Tag) {
	if len(t.Year) > 0 && t.Year != track.TagGetFrame(trackMp3, track.ID3FrameYear) {
		ui.Append("Inflating year metadata...", cui.DebugAppend)
		trackMp3.SetYear(t.Year)
	}
}

func subCondFlushID3FrameFeaturings(t *track.Track, trackMp3 *id3v2.Tag) {
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

func subCondFlushID3FrameTrackNumber(t *track.Track, trackMp3 *id3v2.Tag) {
	if t.TrackNumber > 0 && fmt.Sprintf("%d", t.TrackNumber) != track.TagGetFrame(trackMp3, track.ID3FrameTrackNumber) {
		ui.Append("Inflating track number metadata...", cui.DebugAppend)
		trackMp3.AddFrame(trackMp3.CommonID("Track number/Position in set"),
			id3v2.TextFrame{
				Encoding: id3v2.EncodingUTF8,
				Text:     strconv.Itoa(t.TrackNumber),
			})
	}
}

func subCondFlushID3FrameTrackTotals(t *track.Track, trackMp3 *id3v2.Tag) {
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

func subCondFlushID3FrameArtwork(t *track.Track, trackMp3 *id3v2.Tag) {
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

func subCondFlushID3FrameArtworkURL(t *track.Track, trackMp3 *id3v2.Tag) {
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

func subCondFlushID3FrameOrigin(t *track.Track, trackMp3 *id3v2.Tag) {
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

func subCondFlushID3FrameDuration(t *track.Track, trackMp3 *id3v2.Tag) {
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

func subCondFlushID3FrameSpotifyID(t *track.Track, trackMp3 *id3v2.Tag) {
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

func subCondFlushID3FrameLyrics(t *track.Track, trackMp3 *id3v2.Tag) {
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

	if time.Since(tracksDump.Time) > cacheDuration {
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

func subCondTimestampFlush() {
	if !argDisableTimestampFlush {
		ui.Append("Flushing files timestamps...")
		now := time.Now().Local().Add(time.Duration(-1*len(tracks)) * time.Minute)
		for track := range tracks {
			if !system.FileExists(track.Filename()) {
				continue
			}
			if err := os.Chtimes(track.Filename(), now, now); err != nil {
				ui.Append(fmt.Sprintf("Unable to flush timestamp on %s", track.Filename()), cui.WarningAppend)
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
			pFname   = slug.Make(p.Name)
			pContent string
		)

		if argPlsFile {
			pFname = pFname + ".pls"
		} else {
			pFname = pFname + ".m3u"
		}

		os.Remove(pFname)
		ui.Append(fmt.Sprintf("Creating playlist file at %s...", pFname))
		if system.FileExists(pFname) {
			os.Remove(pFname)
		}

		if argPlsFile {
			pContent = p.PLS(".")
		} else {
			pContent = p.M3U(".")
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

func usrPath() string {
	currentUser, _ := user.Current()
	return fmt.Sprintf("%s/.cache/spotitube", currentUser.HomeDir)
}
