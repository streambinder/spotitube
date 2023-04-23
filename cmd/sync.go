package cmd

import (
	"context"
	"log"
	"os"

	"github.com/arunsworld/nursery"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streambinder/spotitube/downloader"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/processor"
	"github.com/streambinder/spotitube/provider"
	"github.com/streambinder/spotitube/spotify"
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
	spotifyClient     *spotify.Client
	routineSemaphores map[int](chan bool)
	routineQueues     map[int](chan interface{})
	cmdSync           = &cobra.Command{
		Use:   "sync",
		Short: "Synchronize collections",
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				path, _      = cmd.Flags().GetString("path")
				library, _   = cmd.Flags().GetBool("library")
				playlists, _ = cmd.Flags().GetStringArray("playlist")
				albums, _    = cmd.Flags().GetStringArray("album")
				tracks, _    = cmd.Flags().GetStringArray("track")
			)

			if err := os.Chdir(path); err != nil {
				return err
			}

			return nursery.RunConcurrently(
				routineIndex,
				routineAuth,
				routineFetch(library, playlists, albums, tracks),
				routineDecide,
				routineCollect,
				routineProcess,
				routineInstall,
				routineMix,
			)
		},
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			routineSemaphores = map[int](chan bool){
				routineTypeIndex:   make(chan bool, 1),
				routineTypeAuth:    make(chan bool, 1),
				routineTypeInstall: make(chan bool, 1),
			}
			routineQueues = map[int](chan interface{}){
				routineTypeDecide:  make(chan interface{}, 100),
				routineTypeCollect: make(chan interface{}, 100),
				routineTypeProcess: make(chan interface{}, 100),
				routineTypeInstall: make(chan interface{}, 100),
				routineTypeMix:     make(chan interface{}, 100),
			}

			var (
				playlists []string
				albums    []string
				tracks    []string
			)
			playlists, _ = cmd.Flags().GetStringArray("playlist")
			albums, _ = cmd.Flags().GetStringArray("album")
			tracks, _ = cmd.Flags().GetStringArray("track")
			if len(playlists)+len(albums)+len(tracks) == 0 {
				cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
					if f.Name == "library" {
						_ = f.Value.Set("true")
					}
				})
			}
			return
		},
	}
)

func init() {
	cmdRoot.AddCommand(cmdSync)
	cmdSync.Flags().String("path", ".", "Target synchronization path")
	cmdSync.Flags().BoolP("library", "l", false, "Synchronize library (auto-enabled if no collection is supplied)")
	cmdSync.Flags().StringArrayP("playlist", "p", []string{}, "Synchronize playlist")
	cmdSync.Flags().StringArrayP("album", "a", []string{}, "Synchronize album")
	cmdSync.Flags().StringArrayP("track", "t", []string{}, "Synchronize track")
}

// indexer scans a possible local music library
// to be considered as already synchronized
func routineIndex(context.Context, chan error) {
	// remember to signal fetcher
	defer close(routineSemaphores[routineTypeIndex])

	log.Printf("[indexer]\tindexing")
	// TODO: implement indexing
	log.Printf("[indexer]\tindexed")
}

func routineAuth(ctx context.Context, ch chan error) {
	// remember to close auth semaphore
	defer close(routineSemaphores[routineTypeAuth])

	var err error
	spotifyClient, err = spotify.Authenticate()
	if err != nil {
		log.Printf("[auth]\t%s", err)
		routineSemaphores[routineTypeAuth] <- false
		ch <- err
		return
	}

	// once authenticated, signal fetcher
	routineSemaphores[routineTypeAuth] <- true
}

// fetcher pulls data from the upstream
// provider, i.e. Spotify
func routineFetch(library bool, playlists []string, albums []string, tracks []string) func(ctx context.Context, ch chan error) {
	return func(ctx context.Context, ch chan error) {
		// remember to stop passing data to decider and mixer
		defer close(routineQueues[routineTypeDecide])
		defer close(routineQueues[routineTypeMix])
		// block until indexing and authentication is done
		<-routineSemaphores[routineTypeIndex]
		if !<-routineSemaphores[routineTypeAuth] {
			return
		}

		if library {
			if err := spotifyClient.Library(routineQueues[routineTypeDecide]); err != nil {
				ch <- err
				return
			}
		}
		for _, id := range albums {
			if _, err := spotifyClient.Album(id, routineQueues[routineTypeDecide]); err != nil {
				ch <- err
				return
			}
		}
		for _, id := range tracks {
			if _, err := spotifyClient.Track(id, routineQueues[routineTypeDecide]); err != nil {
				ch <- err
				return
			}
		}

		// some special treatment for playlists
		for _, id := range playlists {
			playlist, err := spotifyClient.Playlist(id, routineQueues[routineTypeDecide])
			if err != nil {
				ch <- err
				return
			}
			routineQueues[routineTypeMix] <- playlist
		}
	}
}

// decider finds the right asset to retrieve
// for a given track
func routineDecide(ctx context.Context, ch chan error) {
	// remember to stop passing data to the collector
	// the retriever, the composer and the painter
	defer close(routineQueues[routineTypeCollect])

	cache := make(map[string]bool)
	for event := range routineQueues[routineTypeDecide] {
		track := event.(*entity.Track)

		if _, ok := cache[track.ID]; ok {
			log.Println("[decider]\tignoring duplicate " + track.Title)
			continue
		}

		log.Println("[decider]\t" + track.Title)
		matches, err := provider.Search(track)
		if err != nil {
			ch <- err
			return
		}

		track.UpstreamURL = matches[0].URL
		routineQueues[routineTypeCollect] <- track
		cache[track.ID] = true
	}
}

// collector fetches all the needed assets
// for a blob to be processed (basically
// a wrapper around: retriever, composer and painter)
func routineCollect(ctx context.Context, ch chan error) {
	// remember to stop passing data to installer
	defer close(routineQueues[routineTypeProcess])

	for event := range routineQueues[routineTypeCollect] {
		track := event.(*entity.Track)
		log.Println("[collect]\t" + track.Title)
		if err := nursery.RunConcurrently(
			routineCollectAsset(track),
			routineCollectLyrics(track),
			routineCollectArtwork(track),
		); err != nil {
			log.Printf("[collect]\t%s", err)
			ch <- err
			return
		}
		routineQueues[routineTypeProcess] <- track
	}
}

// retriever pulls a track blob corresponding
// to the (meta)data fetched from upstream
func routineCollectAsset(track *entity.Track) func(context.Context, chan error) {
	return func(ctx context.Context, ch chan error) {
		log.Println("[retriever]\t" + track.Title + " (" + track.UpstreamURL + ")")
		if err := downloader.Download(track.UpstreamURL, track.Path().Download(), nil); err != nil {
			log.Printf("[retriever]\t%s", err)
			ch <- err
			return
		}
	}
}

// composer pulls lyrics to be inserted
// in the fetched blob
func routineCollectLyrics(track *entity.Track) func(context.Context, chan error) {
	return func(ctx context.Context, ch chan error) {
		log.Println("[composer]\t" + track.Title)
		lyrics, err := lyrics.Search(track)
		if err != nil {
			log.Printf("[composer]\t%s", err)
			ch <- err
			return
		}
		track.Lyrics = lyrics
		log.Printf("[composer]\t%d", len(lyrics))
	}
}

// painter pulls image blobs to be inserted
// as artworks in the fetched blob
func routineCollectArtwork(track *entity.Track) func(context.Context, chan error) {
	return func(ctx context.Context, ch chan error) {
		artwork := make(chan []byte, 1)
		defer close(artwork)

		log.Println("[painter]\t" + track.Title + " (" + track.Artwork.URL + ")")
		if err := downloader.Download(track.Artwork.URL, track.Path().Artwork(), processor.Artwork{}, artwork); err != nil {
			log.Printf("[painter]\t%s", err)
			ch <- err
			return
		}

		track.Artwork.Data = <-artwork
		log.Printf("[painter]\t%d", len(track.Artwork.Data))
	}
}

// postprocessor applies some further enhancements
// e.g. combining the downloaded artwork/lyrics
// into the blob
func routineProcess(ctx context.Context, ch chan error) {
	// remember to stop passing data to installer
	defer close(routineQueues[routineTypeInstall])

	for event := range routineQueues[routineTypeProcess] {
		track := event.(*entity.Track)
		log.Println("[postproc]\t" + track.Title)
		if err := processor.Do(track); err != nil {
			log.Printf("[postproc]\t%s", err)
			ch <- err
			return
		}
		routineQueues[routineTypeInstall] <- track
	}
}

// installer move the blob to its final destination
func routineInstall(ctx context.Context, ch chan error) {
	// remember to signal mixer
	defer close(routineSemaphores[routineTypeInstall])

	for event := range routineQueues[routineTypeInstall] {
		track := event.(*entity.Track)
		log.Println("[installer]\t" + track.Title)
	}
}

// mixer wraps playlists to their final destination
func routineMix(context.Context, chan error) {
	// block until installation is done
	<-routineSemaphores[routineTypeInstall]

	for event := range routineQueues[routineTypeMix] {
		playlist := event.(*entity.Playlist)
		log.Println("[mixer]\t" + playlist.Name)
	}
}
