package cmd

import (
	"context"
	"log"

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
	index int = iota
	auth
	decide
	collect
	process
	install
	mix
)

var (
	client     *spotify.Client
	semaphores map[int](chan bool)
	queues     map[int](chan interface{})
	cmdSync    = &cobra.Command{
		Use:   "sync",
		Short: "Synchronize collections",
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				library, _   = cmd.Flags().GetBool("library")
				playlists, _ = cmd.Flags().GetStringArray("playlist")
				albums, _    = cmd.Flags().GetStringArray("album")
				tracks, _    = cmd.Flags().GetStringArray("track")
			)

			return nursery.RunConcurrently(
				indexer,
				authenticator,
				fetcher(library, playlists, albums, tracks),
				decider,
				collector,
				postprocessor,
				installer,
				mixer,
			)
		},
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			semaphores = map[int](chan bool){
				index:   make(chan bool, 1),
				auth:    make(chan bool, 1),
				install: make(chan bool, 1),
			}
			queues = map[int](chan interface{}){
				decide:  make(chan interface{}, 100),
				collect: make(chan interface{}, 100),
				process: make(chan interface{}, 100),
				install: make(chan interface{}, 100),
				mix:     make(chan interface{}, 100),
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
	cmdSync.Flags().BoolP("library", "l", false, "Synchronize library (auto-enabled if no collection is supplied)")
	cmdSync.Flags().StringArrayP("playlist", "p", []string{}, "Synchronize playlist")
	cmdSync.Flags().StringArrayP("album", "a", []string{}, "Synchronize album")
	cmdSync.Flags().StringArrayP("track", "t", []string{}, "Synchronize track")
}

// indexer scans a possible local music library
// to be considered as already synchronized
func indexer(context.Context, chan error) {
	// remember to signal fetcher
	defer close(semaphores[index])

	log.Printf("[indexer]\tindexing")
	// TODO: implement indexing
	log.Printf("[indexer]\tindexed")
}

func authenticator(ctx context.Context, ch chan error) {
	// remember to close auth semaphore
	defer close(semaphores[auth])

	var err error
	client, err = spotify.Authenticate()
	if err != nil {
		log.Printf("[auth]\t%s", err)
		semaphores[auth] <- false
		ch <- err
		return
	}

	// once authenticated, signal fetcher
	semaphores[auth] <- true
}

// fetcher pulls data from the upstream
// provider, i.e. Spotify
func fetcher(library bool, playlists []string, albums []string, tracks []string) func(ctx context.Context, ch chan error) {
	return func(ctx context.Context, ch chan error) {
		// remember to stop passing data to decider and mixer
		defer close(queues[decide])
		defer close(queues[mix])
		// block until indexing and authentication is done
		<-semaphores[index]
		if !<-semaphores[auth] {
			return
		}

		if library {
			if err := client.Library(queues[decide]); err != nil {
				ch <- err
				return
			}
		}
		for _, id := range albums {
			if _, err := client.Album(id, queues[decide]); err != nil {
				ch <- err
				return
			}
		}
		for _, id := range tracks {
			if _, err := client.Track(id, queues[decide]); err != nil {
				ch <- err
				return
			}
		}

		// some special treatment for playlists
		for _, id := range playlists {
			playlist, err := client.Playlist(id, queues[decide])
			if err != nil {
				ch <- err
				return
			}
			queues[mix] <- playlist
		}
	}
}

// decider finds the right asset to retrieve
// for a given track
func decider(ctx context.Context, ch chan error) {
	// remember to stop passing data to the collector
	// the retriever, the composer and the painter
	defer close(queues[collect])

	cache := make(map[string]bool)
	for event := range queues[decide] {
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
		queues[collect] <- track
		cache[track.ID] = true
	}
}

// collector fetches all the needed assets
// for a blob to be processed (basically
// a wrapper around: retriever, composer and painter)
func collector(ctx context.Context, ch chan error) {
	// remember to stop passing data to installer
	defer close(queues[process])

	for event := range queues[collect] {
		track := event.(*entity.Track)
		log.Println("[collect]\t" + track.Title)
		if err := nursery.RunConcurrently(
			retriever(track),
			composer(track),
			painter(track),
		); err != nil {
			log.Printf("[collect]\t%s", err)
			ch <- err
			return
		}
		queues[process] <- track
	}
}

// retriever pulls a track blob corresponding
// to the (meta)data fetched from upstream
func retriever(track *entity.Track) func(context.Context, chan error) {
	return func(ctx context.Context, ch chan error) {
		log.Println("[retriever]\t" + track.Title + " (" + track.UpstreamURL + ")")
		if err := downloader.Download(track.UpstreamURL, track.Path().Download()); err != nil {
			log.Printf("[retriever]\t%s", err)
			ch <- err
			return
		}
	}
}

// painter pulls image blobs to be inserted
// as artworks in the fetched blob
func painter(track *entity.Track) func(context.Context, chan error) {
	return func(ctx context.Context, ch chan error) {
		artwork := make(chan []byte, 1)
		defer close(artwork)

		log.Println("[painter]\t" + track.Title + " (" + track.ArtworkURL + ")")
		if err := downloader.Download(track.ArtworkURL, track.Path().Artwork(), artwork); err != nil {
			log.Printf("[painter]\t%s", err)
			ch <- err
			return
		}

		track.Artwork = <-artwork
		log.Printf("[painter]\t%d", len(track.Artwork))
	}
}

// composer pulls lyrics to be inserted
// in the fetched blob
func composer(track *entity.Track) func(context.Context, chan error) {
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

// postprocessor applies some further enhancements
// e.g. combining the downloaded artwork/lyrics
// into the blob
func postprocessor(ctx context.Context, ch chan error) {
	// remember to stop passing data to installer
	defer close(queues[install])

	for event := range queues[process] {
		track := event.(*entity.Track)
		log.Println("[postproc]\t" + track.Title)
		if err := processor.Do(track); err != nil {
			log.Printf("[postproc]\t%s", err)
			ch <- err
			return
		}
		queues[install] <- track
	}
}

// installer move the blob to its final destination
func installer(context.Context, chan error) {
	// remember to signal mixer
	defer close(semaphores[install])

	for event := range queues[install] {
		track := event.(*entity.Track)
		log.Println("[installer]\t" + track.Title)
	}
}

// mixer wraps playlists to their final destination
func mixer(context.Context, chan error) {
	// block until installation is done
	<-semaphores[install]

	for event := range queues[mix] {
		playlist := event.(*entity.Playlist)
		log.Println("[mixer]\t" + playlist.Name)
	}
}
