package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/adrg/xdg"
	"github.com/arunsworld/nursery"
	"github.com/bogem/id3v2/v2"
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
	"github.com/streambinder/spotitube/util"
	"github.com/streambinder/spotitube/util/anchor"
)

const (
	routineTypeIndex int = iota
	routineTypeAuth
	routineTypeDecide
	routineTypeCollect
	routineTypeProcess
	routineTypeInstall
	routineTypeMix
)

var (
	routineSemaphores map[int](chan bool)
	routineQueues     map[int](chan interface{})
	indexData         = index.New()
	tui               = anchor.New(anchor.Red)
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
			var (
				path             = util.ErrWrap(xdg.UserDirs.Music)(cmd.Flags().GetString("output"))
				playlistEncoding = util.ErrWrap("m3u")(cmd.Flags().GetString("playlist-encoding"))
				manual           = util.ErrWrap(false)(cmd.Flags().GetBool("manual"))
				library          = util.ErrWrap(false)(cmd.Flags().GetBool("library"))
				playlists        = util.ErrWrap([]string{})(cmd.Flags().GetStringArray("playlist"))
				playlistsTracks  = util.ErrWrap([]string{})(cmd.Flags().GetStringArray("playlist-tracks"))
				albums           = util.ErrWrap([]string{})(cmd.Flags().GetStringArray("album"))
				tracks           = util.ErrWrap([]string{})(cmd.Flags().GetStringArray("track"))
				fixes            = util.ErrWrap([]string{})(cmd.Flags().GetStringArray("fix"))
				libraryLimit     = util.ErrWrap(0)(cmd.Flags().GetInt("library-limit"))
			)

			for index, path := range fixes {
				if absPath, err := filepath.Abs(path); err == nil {
					fixes[index] = absPath
				}
			}

			if err := os.Chdir(path); err != nil {
				return err
			}

			return nursery.RunConcurrently(
				routineIndex,
				routineAuth,
				routineFetch(library, playlists, playlistsTracks, albums, tracks, fixes, libraryLimit),
				routineDecide(manual),
				routineCollect,
				routineProcess,
				routineInstall,
				routineMix(playlistEncoding),
			)
		},
		PreRun: func(cmd *cobra.Command, _ []string) {
			routineSemaphores = map[int](chan bool){
				routineTypeIndex:   make(chan bool, 1),
				routineTypeAuth:    make(chan bool, 1),
				routineTypeInstall: make(chan bool, 1),
			}
			routineQueues = map[int](chan interface{}){
				routineTypeDecide:  make(chan interface{}, 10000),
				routineTypeCollect: make(chan interface{}, 10000),
				routineTypeProcess: make(chan interface{}, 10000),
				routineTypeInstall: make(chan interface{}, 10000),
				routineTypeMix:     make(chan interface{}, 10000),
			}

			var (
				playlists       = util.ErrWrap([]string{})(cmd.Flags().GetStringArray("playlist"))
				playlistsTracks = util.ErrWrap([]string{})(cmd.Flags().GetStringArray("playlist-tracks"))
				albums          = util.ErrWrap([]string{})(cmd.Flags().GetStringArray("album"))
				tracks          = util.ErrWrap([]string{})(cmd.Flags().GetStringArray("track"))
				fixes           = util.ErrWrap([]string{})(cmd.Flags().GetStringArray("fix"))
			)
			if len(playlists)+len(playlistsTracks)+len(albums)+len(tracks)+len(fixes) == 0 {
				cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
					if f.Name == "library" {
						util.ErrSuppress(f.Value.Set("true"))
					}
				})
			}
		},
	}
	cmd.Flags().StringP("output", "o", xdg.UserDirs.Music, "Output synchronization path")
	cmd.Flags().String("playlist-encoding", "m3u", "Playlist output files encoding")
	cmd.Flags().BoolP("manual", "m", false, "Enable manual mode (prompts for user-issued URL to use for download")
	cmd.Flags().BoolP("library", "l", false, "Synchronize library (auto-enabled if no collection is supplied)")
	cmd.Flags().StringArrayP("playlist", "p", []string{}, "Synchronize playlist")
	cmd.Flags().StringArray("playlist-tracks", []string{}, "Synchronize playlist tracks without playlist file")
	cmd.Flags().StringArrayP("album", "a", []string{}, "Synchronize album")
	cmd.Flags().StringArrayP("track", "t", []string{}, "Synchronize track")
	cmd.Flags().StringArrayP("fix", "f", []string{}, "Fix local track")
	cmd.Flags().Int("library-limit", 0, "Number of tracks to fetch from library (unlimited if 0)")
	return cmd
}

// indexer scans a possible local music library
// to be considered as already synchronized
func routineIndex(_ context.Context, ch chan error) {
	// remember to signal fetcher
	defer close(routineSemaphores[routineTypeIndex])

	tui.Lot("index").Printf("scanning")
	if err := indexData.Build("."); err != nil {
		tui.Printf("indexing failed: %s", err)
		routineSemaphores[routineTypeIndex] <- false
		ch <- err
		return
	}
	tui.Lot("index").Close(strconv.Itoa(indexData.Size()) + " tracks")

	// once indexed, sidgnal fetcher
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
	tui.Lot("auth").Close()

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

		fetched := make(chan interface{}, 10000)
		defer close(fetched)
		go func() {
			counter := 0
			for event := range fetched {
				counter++
				track := event.(*entity.Track)
				tui.Lot("fetch").Printf("%s by %s", track.Title, track.Artists[0])
			}
			tui.Lot("fetch").Close(fmt.Sprintf("%d tracks", counter))
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
		if err := routineFetchPlaylists(append(playlists, playlistsTracks...), fetched); err != nil {
			ch <- err
			return
		}
	}
}

func routineFetchFixesIDs(fixes []string) ([]string, error) {
	var localTracks []string
	for _, path := range fixes {
		tui.Lot("fetch").Printf("track %s", path)
		tag, err := id3.Open(path, id3v2.Options{Parse: true})
		if err != nil {
			return nil, err
		}

		id := tag.SpotifyID()
		if len(id) == 0 {
			return nil, errors.New("track " + path + " does not have spotify ID metadata set")
		}

		localTracks = append(localTracks, id)
		indexData.SetPath(path, index.Flush)
		if err := tag.Close(); err != nil {
			return nil, err
		}
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

func routineFetchPlaylists(playlists []string, fetched chan interface{}) error {
	for index, id := range playlists {
		tui.Lot("fetch").Printf("playlist %s", id)
		playlist, err := spotifyClient.Playlist(id, routineQueues[routineTypeDecide], fetched)
		if err != nil {
			return err
		}
		if index < len(playlists) {
			routineQueues[routineTypeMix] <- playlist
		}
	}
	return nil
}

// decider finds the right asset to retrieve
// for a given track
func routineDecide(manualMode bool) func(context.Context, chan error) {
	return func(_ context.Context, ch chan error) {
		// remember to stop passing data to the collector
		// the retriever, the composer and the painter
		defer close(routineQueues[routineTypeCollect])

		for event := range routineQueues[routineTypeDecide] {
			track := event.(*entity.Track)

			if status, ok := indexData.Get(track); !ok {
				tui.Printf("sync %s by %s", track.Title, track.Artists[0])
				indexData.Set(track, index.Online)
			} else if status == index.Online {
				tui.Printf("skip %s by %s", track.Title, track.Artists[0])
				continue
			} else if status == index.Offline {
				continue
			}

			if manualMode {
				tui.Lot("decide").Printf("waiting on user input")
				track.UpstreamURL = tui.Reads("URL for %s by %s:", track.Title, track.Artists[0])
				tui.Lot("decide").Wipe()
				if len(track.UpstreamURL) == 0 {
					continue
				}
			} else {
				tui.Lot("decide").Printf("%s by %s", track.Title, track.Artists[0])
				matches, err := provider.Search(track)
				tui.Lot("decide").Wipe()
				if err != nil {
					ch <- err
					return
				}

				if len(matches) == 0 {
					tui.AnchorPrintf("%s by %s (id: %s) not found", track.Title, track.Artists[0], track.ID)
					continue
				}
				track.UpstreamURL = matches[0].URL
			}
			routineQueues[routineTypeCollect] <- track
		}
		tui.Lot("decide").Close()
	}
}

// collector fetches all the needed assets
// for a blob to be processed (basically
// a wrapper around: retriever, composer and painter)
func routineCollect(_ context.Context, ch chan error) {
	// remember to stop passing data to installer
	defer close(routineQueues[routineTypeProcess])

	for event := range routineQueues[routineTypeCollect] {
		track := event.(*entity.Track)
		if err := nursery.RunConcurrently(
			routineCollectAsset(track),
			routineCollectLyrics(track),
			routineCollectArtwork(track),
		); err != nil {
			ch <- err
			return
		}
		routineQueues[routineTypeProcess] <- track
	}
	tui.Lot("download").Close()
	tui.Lot("compose").Close()
	tui.Lot("paint").Close()
}

// retriever pulls a track blob corresponding
// to the (meta)data fetched from upstream
func routineCollectAsset(track *entity.Track) func(context.Context, chan error) {
	return func(_ context.Context, ch chan error) {
		tui.Lot("download").Print(track.UpstreamURL)
		if err := downloader.Download(track.UpstreamURL, track.Path().Download(), nil); err != nil {
			tui.AnchorPrintf("download failure: %s", err)
			ch <- err
			return
		}
		tui.Printf("asset for %s by %s: %s", track.Title, track.Artists[0], track.UpstreamURL)
		tui.Lot("download").Wipe()
	}
}

// composer pulls lyrics to be inserted
// in the fetched blob
func routineCollectLyrics(track *entity.Track) func(context.Context, chan error) {
	return func(_ context.Context, ch chan error) {
		tui.Lot("compose").Printf("%s by %s", track.Title, track.Artists[0])
		lyrics, err := lyrics.Search(track)
		if err != nil {
			tui.AnchorPrintf("compose failure: %s", err)
			ch <- err
			return
		}
		tui.Lot("compose").Wipe()
		track.Lyrics = lyrics
		tui.Printf("lyrics for %s by %s: %s", track.Title, track.Artists[0], util.Excerpt(lyrics))
	}
}

// painter pulls image blobs to be inserted
// as artworks in the fetched blob
func routineCollectArtwork(track *entity.Track) func(context.Context, chan error) {
	return func(_ context.Context, ch chan error) {
		artwork := make(chan []byte, 1)
		defer close(artwork)

		tui.Lot("paint").Printf("%s by %s", track.Title, track.Artists[0])
		if err := downloader.Download(track.Artwork.URL, track.Path().Artwork(), processor.Artwork{}, artwork); err != nil {
			tui.AnchorPrintf("compose failure: %s", err)
			ch <- err
			return
		}

		tui.Lot("paint").Wipe()
		track.Artwork.Data = <-artwork
		tui.Printf("artwork for %s by %s: %s", track.Title, track.Artists[0], util.HumanizeBytes(len(track.Artwork.Data)))
	}
}

// postprocessor applies some further enhancements
// e.g. combining the downloaded artwork/lyrics
// into the blob
func routineProcess(_ context.Context, ch chan error) {
	// remember to stop passing data to installer
	defer close(routineQueues[routineTypeInstall])

	for event := range routineQueues[routineTypeProcess] {
		track := event.(*entity.Track)
		tui.Lot("process").Printf("%s by %s", track.Title, track.Artists[0])
		if err := processor.Do(track); err != nil {
			tui.AnchorPrintf("processing failed for %s by %s: %s", track.Title, track.Artists[0], err)
			ch <- err
			return
		}
		tui.Lot("process").Wipe()
		routineQueues[routineTypeInstall] <- track
	}
	tui.Lot("process").Close()
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
		tui.Lot("install").Printf("%s by %s ", track.Title, track.Artists[0])
		if err := util.FileMoveOrCopy(track.Path().Download(), track.Path().Final(), status == index.Flush); err != nil {
			tui.AnchorPrintf("installation failed for %s by %s: %s", track.Title, track.Artists[0], err)
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
	return func(_ context.Context, ch chan error) {
		// block until installation is done
		<-routineSemaphores[routineTypeInstall]

		counter := 0
		for event := range routineQueues[routineTypeMix] {
			counter++
			playlist := event.(*playlist.Playlist)
			tui.Lot("mix").Printf("%s", playlist.Name)
			encoder, err := playlist.Encoder(encoding)
			if err != nil {
				tui.AnchorPrintf("mixing failed for %s: %s", playlist.Name, err)
				ch <- err
				return
			}

			for _, track := range playlist.Tracks {
				if trackStatus, ok := indexData.Get(track); !ok || (trackStatus != index.Installed && trackStatus != index.Offline) {
					continue
				}

				if err := encoder.Add(track); err != nil {
					tui.AnchorPrintf("adding track to %s failed: %s", playlist.Name, err)
					ch <- err
					return

				}
			}

			if err := encoder.Close(); err != nil {
				tui.AnchorPrintf("closing playlist %s failed: %s", playlist.Name, err)
				ch <- err
				return
			}
		}
		tui.Lot("mix").Close(fmt.Sprintf("%d playlists", counter))
	}
}
