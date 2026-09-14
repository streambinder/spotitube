package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/adrg/xdg"
	"github.com/arunsworld/nursery"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streambinder/spotitube/downloader"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/entity/index"
	"github.com/streambinder/spotitube/entity/playlist"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/processor"
	"github.com/streambinder/spotitube/provider"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
	"github.com/streambinder/spotitube/sys/anchor"
	commands "github.com/streambinder/spotitube/sys/cmd"
)

const (
	routineTypeIndex int = iota
	routineTypeAuth
	routineTypeDecide
	routineTypeCollect
	routineTypeProcess
	routineTypeInstall
	routineTypeMix

	// pipeline buffer depth — large enough to decouple stages so the
	// fetcher can push ahead while slower stages (provider search,
	// download) catch up, without being an unbounded memory commitment
	pipelineBuffer = 4096

	// consecutive search failures triggering the decide circuit breaker
	maxConsecutiveFailures = 3
)

var (
	routineSemaphores map[int](chan bool)
	routineQueues     map[int](chan interface{})
	indexData         = index.New()
	tui               = anchor.New(anchor.Red)

	// serializes the decide index check-and-claim across workers:
	// without it, two workers could both clear the collision check
	// for same-filename tracks and defer the failure to install
	decideIndexMu sync.Mutex
	// parallel decide workers in automatic mode; deliberately small,
	// YouTube searches are additionally capped inside the provider
	decideWorkers = 4
	// parallel collect workers; downloads are network-bound, so tracks
	// overlap while each keeps asset/lyrics/artwork concurrent
	collectWorkers = 4
	// parallel process workers; ffmpeg re-encoding is CPU-bound, so a few
	// tracks overlap while each keeps its internal pipeline serial
	processWorkers = 4
)

func init() {
	cmdRoot.AddCommand(cmdSync())
}

func cmdSync() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "sync",
		Short:        "Synchronize collections",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := commands.ValidateEnvironment(); err != nil {
				return err
			}

			var (
				path             = sys.ErrWrap(xdg.UserDirs.Music)(cmd.Flags().GetString("output"))
				playlistEncoding = sys.ErrWrap("m3u")(cmd.Flags().GetString("playlist-encoding"))
				manual           = sys.ErrWrap(false)(cmd.Flags().GetBool("manual"))
				library          = sys.ErrWrap(false)(cmd.Flags().GetBool("library"))
				playlists        = sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("playlist"))
				playlistsTracks  = sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("playlist-tracks"))
				albums           = sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("album"))
				tracks           = sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("track"))
				fixes            = sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("fix"))
				libraryLimit     = sys.ErrWrap(0)(cmd.Flags().GetInt("library-limit"))
				plain            = sys.ErrWrap(false)(cmd.Flags().GetBool("plain"))
				skipLyrics       = sys.ErrWrap(false)(cmd.Flags().GetBool("skip-lyrics"))
			)

			if plain {
				tui.EnablePlainMode()
			}

			for index, path := range fixes {
				absPath, absErr := filepath.Abs(path)
				fixes[index] = sys.Ternary(absErr == nil, absPath, path)
			}

			if err := os.Chdir(path); err != nil {
				return err
			}

			if err := nursery.RunConcurrently(
				routineIndex,
				routineAuth,
				routineFetch(library, playlists, playlistsTracks, albums, tracks, fixes, libraryLimit),
				routineDecide(manual),
				routineCollect(skipLyrics),
				routineProcess,
				routineInstall,
				routineMix(playlistEncoding),
			); err != nil {
				return err
			}

			tui.Printf("synchronization complete")
			return nil
		},
		PreRun: func(cmd *cobra.Command, _ []string) {
			routineSemaphores = map[int](chan bool){
				routineTypeIndex:   make(chan bool, 1),
				routineTypeAuth:    make(chan bool, 1),
				routineTypeInstall: make(chan bool, 1),
			}
			routineQueues = map[int](chan interface{}){
				routineTypeDecide:  make(chan interface{}, pipelineBuffer),
				routineTypeCollect: make(chan interface{}, pipelineBuffer),
				routineTypeProcess: make(chan interface{}, pipelineBuffer),
				routineTypeInstall: make(chan interface{}, pipelineBuffer),
				routineTypeMix:     make(chan interface{}, pipelineBuffer),
			}

			var (
				playlists       = sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("playlist"))
				playlistsTracks = sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("playlist-tracks"))
				albums          = sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("album"))
				tracks          = sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("track"))
				fixes           = sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("fix"))
			)
			if len(playlists)+len(playlistsTracks)+len(albums)+len(tracks)+len(fixes) == 0 {
				cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
					if f.Name == "library" {
						sys.ErrSuppress(f.Value.Set("true"))
					}
				})
			}
		},
	}
	cmd.Flags().StringP("output", "o", xdg.UserDirs.Music, "Output synchronization path")
	cmd.Flags().String("playlist-encoding", "m3u", "Playlist output files encoding")
	cmd.Flags().BoolP("manual", "m", false, "Enable manual mode (prompts for user-issued URL to use for download)")
	cmd.Flags().BoolP("library", "l", false, "Synchronize library (auto-enabled if no collection is supplied)")
	cmd.Flags().StringArrayP("playlist", "p", []string{}, "Synchronize playlist")
	cmd.Flags().StringArray("playlist-tracks", []string{}, "Synchronize playlist tracks without playlist file")
	cmd.Flags().StringArrayP("album", "a", []string{}, "Synchronize album")
	cmd.Flags().StringArrayP("track", "t", []string{}, "Synchronize track")
	cmd.Flags().StringArrayP("fix", "f", []string{}, "Fix local track")
	cmd.Flags().Int("library-limit", 0, "Number of tracks to fetch from library (unlimited if 0)")
	cmd.Flags().Bool("plain", false, "Enable plain mode (no fancy TUI anchored output)")
	cmd.Flags().Bool("skip-lyrics", false, "Skip lyrics collection")
	return cmd
}

// indexer scans a possible local music library
// to be considered as already synchronized
func routineIndex(_ context.Context, ch chan error) {
	// remember to signal fetcher
	defer close(routineSemaphores[routineTypeIndex])

	indexed := make(chan string)
	var indexCounter sync.WaitGroup
	indexCounter.Add(1)
	go func() {
		defer indexCounter.Done()
		counter := 0
		for path := range indexed {
			counter++
			tui.Lot("index").Printf("%s", filepath.Base(path))
		}
		tui.Lot("index").Close(strconv.Itoa(counter) + " tracks")
	}()
	defer func() {
		close(indexed)
		indexCounter.Wait()
	}()

	tui.Lot("index").Printf("scanning")
	if err := indexData.BuildWithProgress(".", indexed); err != nil {
		tui.Printf("indexing failed: %s", err)
		routineSemaphores[routineTypeIndex] <- false
		ch <- err
		return
	}

	// once indexed, signal fetcher
	routineSemaphores[routineTypeIndex] <- true
}

func routineAuth(_ context.Context, ch chan error) {
	// remember to close auth semaphore
	defer close(routineSemaphores[routineTypeAuth])

	tui.Lot("auth").Printf("authenticating")
	var err error
	spotifyClient, err = spotify.Authenticate(spotify.BrowserProcessor)
	if err != nil {
		tui.Printf("authentication failed: %s", err)
		routineSemaphores[routineTypeAuth] <- false
		ch <- err
		return
	}
	defer spotifyClient.Close()
	username, err := spotifyClient.Username()
	if err != nil {
		tui.Printf("could not resolve authenticated username: %s", err)
		tui.Lot("auth").Close()
	} else {
		tui.Lot("auth").Close(username)
	}

	// once authenticated, signal fetcher
	routineSemaphores[routineTypeAuth] <- true
}

// fetcher pulls data from the upstream
// provider, i.e. Spotify
func routineFetch(library bool, playlists, playlistsTracks, albums, tracks, fixes []string, libraryLimit int) func(ctx context.Context, ch chan error) {
	return func(_ context.Context, ch chan error) {
		// remember to stop passing data to decider and mixer
		defer close(routineQueues[routineTypeDecide])
		defer close(routineQueues[routineTypeMix])
		// block until indexing and authentication is done
		if !<-routineSemaphores[routineTypeIndex] {
			return
		}
		if !<-routineSemaphores[routineTypeAuth] {
			return
		}

		fetched := make(chan interface{})
		var fetchCounter sync.WaitGroup
		fetchCounter.Add(1)
		go func() {
			defer fetchCounter.Done()
			counter := 0
			for event := range fetched {
				counter++
				track := event.(*entity.Track)
				tui.Lot("fetch").Printf("%s by %s", track.Title, track.Artist())
			}
			tui.Lot("fetch").Close(fmt.Sprintf("%d tracks", counter))
		}()
		defer func() {
			close(fetched)
			fetchCounter.Wait()
		}()

		fixesTracks, fixesErr := routineFetchFixesIDs(fixes)
		if fixesErr != nil {
			ch <- fixesErr
			return
		}
		tracks = append(tracks, fixesTracks...)

		if err := routineFetchLibrary(library, libraryLimit, fetched); err != nil {
			ch <- err
			return
		}
		if err := routineFetchAlbums(albums, fetched); err != nil {
			ch <- err
			return
		}
		if err := routineFetchTracks(tracks, fetched); err != nil {
			ch <- err
			return
		}
		if err := routineFetchPlaylists(append(playlists, playlistsTracks...), len(playlists), fetched); err != nil {
			ch <- err
			return
		}
	}
}

func routineFetchFixesIDs(fixes []string) ([]string, error) {
	var localTracks []string
	for _, path := range fixes {
		tui.Lot("fetch").Printf("track %s", path)
		tag, err := id3.OpenSpotifyID(path)
		if err != nil {
			return nil, err
		}

		id := tag.SpotifyID()
		if err := tag.Close(); err != nil {
			return nil, err
		}
		if len(id) == 0 {
			return nil, errors.New("track " + path + " does not have spotify ID metadata set")
		}

		// the track is about to be re-downloaded to its canonical path:
		// drop the stale original, otherwise it would linger next to the
		// fresh download as a permanent duplicate
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		localTracks = append(localTracks, id)
		indexData.SetID(id, index.Flush)
	}
	return localTracks, nil
}

func routineFetchLibrary(library bool, libraryLimit int, fetched chan interface{}) error {
	if !library {
		return nil
	}

	tui.Lot("fetch").Printf("library")
	return spotifyClient.Library(libraryLimit, routineQueues[routineTypeDecide], fetched)
}

func routineFetchAlbums(albums []string, fetched chan interface{}) error {
	for _, id := range albums {
		tui.Lot("fetch").Printf("album %s", id)
		if _, err := spotifyClient.Album(id, routineQueues[routineTypeDecide], fetched); err != nil {
			return err
		}
	}
	return nil
}

func routineFetchTracks(tracks []string, fetched chan interface{}) error {
	for _, id := range tracks {
		tui.Lot("fetch").Printf("track %s", id)
		if _, err := spotifyClient.Track(id, routineQueues[routineTypeDecide], fetched); err != nil {
			return err
		}
	}
	return nil
}

func routineFetchPlaylists(playlists []string, playlistsWithFile int, fetched chan interface{}) error {
	for index, id := range playlists {
		tui.Lot("fetch").Printf("playlist %s", id)
		playlist, err := spotifyClient.Playlist(id, routineQueues[routineTypeDecide], fetched)
		if err != nil {
			return err
		}
		if index < playlistsWithFile {
			routineQueues[routineTypeMix] <- playlist
		}
	}
	return nil
}

// decider finds the right asset to retrieve
// for a given track
func routineDecide(manualMode bool) func(context.Context, chan error) {
	return func(ctx context.Context, ch chan error) {
		// remember to stop passing data to the collector
		// the retriever, the composer and the painter
		defer close(routineQueues[routineTypeCollect])

		// terminal prompts can't be parallelized: manual mode stays serial
		if manualMode {
			decideManual(ch)
			return
		}
		decideParallel(ctx, ch)
	}
}

// decideManual is the original single-consumer loop,
// kept for manual mode where the user is prompted per track
func decideManual(ch chan error) {
	for event := range routineQueues[routineTypeDecide] {
		track := event.(*entity.Track)

		proceed, fatal := decideClassify(ch, track)
		if fatal {
			return
		}
		if !proceed {
			continue
		}

		tui.Lot("decide").Printf("waiting on user input")
		track.UpstreamURL = tui.Reads("URL for %s by %s:", track.Title, track.Artist())
		tui.Lot("decide").Wipe()
		if len(track.UpstreamURL) == 0 {
			continue
		}
		routineQueues[routineTypeCollect] <- track
	}
	tui.Lot("decide").Close()
}

func decideParallel(_ context.Context, ch chan error) {
	// consecutive search failures trigger a circuit breaker:
	// if all providers fail repeatedly, stop wasting time retrying
	var consecutiveFailures atomic.Int32

	if err := nursery.RunMultipleCopiesConcurrently(decideWorkers, func(ctx context.Context, ch chan error) {
		decideWorker(ctx, ch, &consecutiveFailures)
	}); err != nil {
		decide := routineQueues[routineTypeDecide]
		go func() {
			for {
				if _, ok := <-decide; !ok {
					return
				}
			}
		}()
		ch <- err
		return
	}
	tui.Lot("decide").Close()
}

// decideWorker consumes the decide queue until it is closed,
// the context is cancelled or a fatal collision aborts the sync
func decideWorker(ctx context.Context, ch chan error, consecutiveFailures *atomic.Int32) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-routineQueues[routineTypeDecide]:
			if !ok {
				return
			}
			if fatal := decideTrack(ch, consecutiveFailures, event.(*entity.Track)); fatal {
				return
			}
		}
	}
}

// decideClassify runs the index check-and-claim for a track and reports
// whether it should proceed to search; fatal is true when the whole sync
// must abort on a filename collision
func decideClassify(ch chan error, track *entity.Track) (proceed, fatal bool) {
	decideIndexMu.Lock()
	defer decideIndexMu.Unlock()

	var idStatus int
	idKnown := false
	if len(track.ID) > 0 {
		idStatus, idKnown = indexData.GetID(track.ID)
	}
	_, pathKnown := indexData.GetPath(track.Path().Final())

	switch {
	case !idKnown && pathKnown:
		msg := fmt.Sprintf("filename collision: %q would be shared by %q by %q (spotify id %s) and another track with the same artist and title: rename or drop one of them and re-run",
			track.Path().Final(), track.Title, track.Artist(), track.ID)
		tui.AnchorPrintf("%s", msg)
		ch <- errors.New(msg)
		return false, true
	case !idKnown:
		tui.Printf("sync %s by %s", track.Title, track.Artist())
		indexData.Set(track, index.Online)
		return true, false
	case idStatus == index.Online:
		tui.Printf("skip %s by %s", track.Title, track.Artist())
		return false, false
	case idStatus == index.Offline:
		return false, false
	default:
		// Flush / Installed: re-sync
		return true, false
	}
}

// decideTrack is the automatic-mode per-track pipeline run by each worker
func decideTrack(ch chan error, consecutiveFailures *atomic.Int32, track *entity.Track) (fatal bool) {
	proceed, fatal := decideClassify(ch, track)
	if fatal || !proceed {
		return fatal
	}

	if consecutiveFailures.Load() >= maxConsecutiveFailures {
		tui.AnchorPrintf("%s by %s (id: %s) skipped: search unavailable", track.Title, track.Artist(), track.ID)
		return false
	}

	matches, err := provider.Search(track)
	if err != nil {
		consecutiveFailures.Add(1)
		tui.AnchorPrintf("%s by %s (id: %s) search failed: %v", track.Title, track.Artist(), track.ID, err)
		return false
	}

	consecutiveFailures.Store(0)
	if len(matches) == 0 {
		tui.AnchorPrintf("%s by %s (id: %s) not found", track.Title, track.Artist(), track.ID)
		return false
	}
	track.UpstreamURL = matches[0].URL
	routineQueues[routineTypeCollect] <- track
	return false
}

// collector fetches all the needed assets
// for a blob to be processed (basically
// a wrapper around: retriever, composer and painter)
func routineCollect(skipLyrics bool) func(context.Context, chan error) {
	return func(_ context.Context, _ chan error) {
		// remember to stop passing data to the processor
		defer close(routineQueues[routineTypeProcess])

		// workers never report errors back: a failed collection
		// drops the track without aborting the sync
		_ = nursery.RunMultipleCopiesConcurrently(collectWorkers, func(ctx context.Context, _ chan error) { //nolint:errcheck
			collectWorker(ctx, skipLyrics)
		})

		tui.Lot("download").Close()
		tui.Lot("compose").Close()
		tui.Lot("paint").Close()
	}
}

// collectWorker consumes the collect queue until it is closed or the
// context is cancelled, running the per-track collection pipeline
func collectWorker(ctx context.Context, skipLyrics bool) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-routineQueues[routineTypeCollect]:
			if !ok {
				return
			}
			collectTrack(skipLyrics, event.(*entity.Track))
		}
	}
}

// collectTrack downloads asset, lyrics and artwork concurrently for a single
// track, then hands it to the processor; a failed collection drops the track
// without aborting the sync
func collectTrack(skipLyrics bool, track *entity.Track) {
	routines := []nursery.ConcurrentJob{routineCollectAsset(track)}
	if !skipLyrics {
		routines = append(routines, routineCollectLyrics(track))
	}
	routines = append(routines, routineCollectArtwork(track))
	if err := nursery.RunConcurrently(routines...); err != nil {
		tui.AnchorPrintf("%s by %s (id: %s) collection failed: %v", track.Title, track.Artist(), track.ID, err)
		return
	}
	routineQueues[routineTypeProcess] <- track
}

// retriever pulls a track blob corresponding
// to the (meta)data fetched from upstream
func routineCollectAsset(track *entity.Track) func(context.Context, chan error) {
	return func(ctx context.Context, ch chan error) {
		tui.Lot("download").Print(track.UpstreamURL)
		if err := downloader.Download(ctx, track.UpstreamURL, track.Path().Download(), nil); err != nil {
			tui.AnchorPrintf("download failure: %s", err)
			ch <- err
			return
		}
		tui.Printf("asset for %s by %s: %s", track.Title, track.Artist(), track.UpstreamURL)
		tui.Lot("download").Wipe()
	}
}

// composer pulls lyrics to be inserted
// in the fetched blob
func routineCollectLyrics(track *entity.Track) func(context.Context, chan error) {
	return func(_ context.Context, _ chan error) {
		tui.Lot("compose").Printf("%s by %s", track.Title, track.Artist())
		lyrics, err := lyrics.Search(track)
		if err != nil {
			// lyrics are a nice-to-have: a failure must not
			// discard a track with audio and artwork ready
			tui.AnchorPrintf("lyrics for %s by %s failed: %s", track.Title, track.Artist(), err)
		}
		tui.Lot("compose").Wipe()
		track.Lyrics = lyrics
		tui.Printf("lyrics for %s by %s: %s", track.Title, track.Artist(), sys.Fallback(sys.Excerpt(sys.FirstLine(lyrics), 64), "not found"))
	}
}

// painter pulls image blobs to be inserted
// as artworks in the fetched blob
func routineCollectArtwork(track *entity.Track) func(context.Context, chan error) {
	return func(ctx context.Context, ch chan error) {
		// an empty artwork URL would deadlock below: downloader.Download
		// returns immediately without feeding the channel, so there is
		// nothing to wait for
		if len(track.Artwork.URL) == 0 {
			tui.Printf("artwork for %s by %s: %s", track.Title, track.Artist(), "not found")
			return
		}

		artwork := make(chan []byte, 1)
		defer close(artwork)

		tui.Lot("paint").Printf("%s by %s", track.Title, track.Artist())
		if err := downloader.Download(ctx, track.Artwork.URL, track.Path().Artwork(), processor.Artwork{}, artwork); err != nil {
			tui.AnchorPrintf("compose failure: %s", err)
			ch <- err
			return
		}

		tui.Lot("paint").Wipe()
		track.Artwork.Data = <-artwork
		tui.Printf("artwork for %s by %s: %s", track.Title, track.Artist(), sys.HumanizeBytes(len(track.Artwork.Data)))
	}
}

// postprocessor applies some further enhancements
// e.g. combining the downloaded artwork/lyrics
// into the blob
func routineProcess(_ context.Context, ch chan error) {
	// remember to stop passing data to installer
	defer close(routineQueues[routineTypeInstall])

	if err := nursery.RunMultipleCopiesConcurrently(processWorkers, func(ctx context.Context, ch chan error) {
		processWorker(ctx, ch)
	}); err != nil {
		ch <- err
		return
	}

	tui.Lot("process").Close()
}

// processWorker consumes the process queue until it is closed or the
// context is cancelled; a failed track aborts the whole sync
func processWorker(ctx context.Context, ch chan error) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-routineQueues[routineTypeProcess]:
			if !ok {
				return
			}
			track := event.(*entity.Track)
			tui.Lot("process").Printf("%s by %s", track.Title, track.Artist())
			if err := processor.Do(track); err != nil {
				if errors.Is(err, processor.ErrLoudnessSkipped) {
					tui.AnchorPrintf("loudness normalization skipped for %s by %s: %s", track.Title, track.Artist(), errors.Unwrap(err))
				} else {
					tui.AnchorPrintf("processing failed for %s by %s: %s", track.Title, track.Artist(), err)
					ch <- err
					return
				}
			}
			tui.Lot("process").Wipe()
			routineQueues[routineTypeInstall] <- track
		}
	}
}

// installer move the blob to its final destination
func routineInstall(_ context.Context, ch chan error) {
	// remember to signal mixer
	defer close(routineSemaphores[routineTypeInstall])

	for event := range routineQueues[routineTypeInstall] {
		var (
			track     = event.(*entity.Track)
			status, _ = indexData.Get(track)
		)
		tui.Lot("install").Printf("%s by %s ", track.Title, track.Artist())
		if err := sys.FileMoveOrCopy(track.Path().Download(), track.Path().Final(), status == index.Flush); err != nil {
			tui.AnchorPrintf("installation failed for %s by %s: %s", track.Title, track.Artist(), err)
			ch <- err
			return
		}
		tui.Lot("install").Wipe()
		indexData.Set(track, index.Installed)
	}
	tui.Lot("install").Close(strconv.Itoa(indexData.Size(index.Installed)) + " tracks")
}

// mixer wraps playlists to their final destination
func routineMix(encoding string) func(context.Context, chan error) {
	return func(_ context.Context, _ chan error) {
		// block until installation is done
		<-routineSemaphores[routineTypeInstall]

		counter := 0
		for event := range routineQueues[routineTypeMix] {
			counter++
			playlist := event.(*playlist.Playlist)
			tui.Lot("mix").Printf("%s", playlist.Name)
			encoder, err := playlist.Encoder(encoding)
			if err != nil {
				// a broken playlist must not kill the run after
				// everything has been downloaded: skip it instead
				tui.AnchorPrintf("mixing failed for %s: %s", playlist.Name, err)
				continue
			}

			for _, track := range playlist.Tracks {
				if trackStatus, ok := indexData.Get(track); !ok || (trackStatus != index.Installed && trackStatus != index.Offline) {
					tui.Printf("skipping %s by %s in playlist %s: not installed", track.Title, track.Artist(), playlist.Name)
					continue
				}

				if err := encoder.Add(track); err != nil {
					// stop feeding a failed encoder, keep what was added
					tui.AnchorPrintf("adding track to %s failed: %s", playlist.Name, err)
					break

				}
			}

			if err := encoder.Close(); err != nil {
				tui.AnchorPrintf("closing playlist %s failed: %s", playlist.Name, err)
				continue
			}
		}
		tui.Lot("mix").Close(fmt.Sprintf("%d playlists", counter))
	}
}
