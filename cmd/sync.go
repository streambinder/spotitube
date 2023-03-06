package cmd

import (
	"context"
	"log"

	"github.com/arunsworld/nursery"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/spotify"
)

const (
	index int = iota
	authenticate
	decide
	download
	paint
	compose
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
				downloader,
				painter,
				composer,
				processor,
				installer,
				mixer,
			)
		},
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			semaphores = map[int](chan bool){
				index:        make(chan bool, 1),
				authenticate: make(chan bool, 1),
				mix:          make(chan bool, 1),
			}
			queues = map[int](chan interface{}){
				decide:   make(chan interface{}),
				download: make(chan interface{}),
				paint:    make(chan interface{}),
				compose:  make(chan interface{}),
				process:  make(chan interface{}),
				install:  make(chan interface{}),
				mix:      make(chan interface{}),
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
	// remember to close authenticate semaphore
	defer close(semaphores[authenticate])

	var err error
	client, err = spotify.Authenticate()
	if err != nil {
		log.Printf("[auth]\t%s", err)
		semaphores[authenticate] <- false
		ch <- err
		return
	}

	// once authenticated, signal fetcher
	semaphores[authenticate] <- true
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
		if !<-semaphores[authenticate] {
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

// decider finds the right asset to download
// for a given track
func decider(context.Context, chan error) {
	// remember to stop passing data to the downloader
	defer close(queues[download])

	cache := make(map[string]bool)
	for event := range queues[decide] {
		track := event.(*entity.Track)

		if _, ok := cache[track.ID]; ok {
			log.Println("[decider]\tignoring duplicate " + track.Title)
			continue
		}

		log.Println("[decider]\t" + track.Title)
		track.UpstreamURL = "http://whatev.er/blob.mp3"
		queues[download] <- track
		cache[track.ID] = true
	}
}

// downloader pulls a track blob corresponding
// to the (meta)data fetched from upstream
func downloader(context.Context, chan error) {
	// remember to stop passing data to painter
	defer close(queues[paint])

	for event := range queues[download] {
		track := event.(*entity.Track)
		log.Println("[download]\t" + track.Title)
		queues[paint] <- track
	}
}

// painter pulls image blobs to be inserted
// as artworks in the fetched blob
func painter(context.Context, chan error) {
	// remember to stop passing data to composer
	defer close(queues[compose])

	for event := range queues[paint] {
		track := event.(*entity.Track)
		log.Println("[painter]\t" + track.Title)
		queues[compose] <- track
	}
}

// composer pulls lyrics to be inserted
// in the fetched blob
func composer(context.Context, chan error) {
	// remember to stop passing data to processor
	defer close(queues[process])

	for event := range queues[compose] {
		track := event.(*entity.Track)
		log.Println("[composer]\t" + track.Title)
		queues[process] <- track
	}
}

// processor applies some further enhancements
// e.g. combining the downloaded artwork/lyrics
// into the blob
func processor(context.Context, chan error) {
	// remember to stop passing data to installer
	defer close(queues[install])

	for event := range queues[process] {
		track := event.(*entity.Track)
		log.Println("[processor]\t" + track.Title)
		queues[install] <- track
	}
}

// installer move the blob to its final destination
func installer(context.Context, chan error) {
	// remember to signal mixer
	defer close(semaphores[mix])

	for event := range queues[install] {
		track := event.(*entity.Track)
		log.Println("[installer]\t" + track.Title)
	}
}

// mixer wraps playlists to their final destination
func mixer(context.Context, chan error) {
	// block until installation is done
	<-semaphores[index]

	for event := range queues[mix] {
		playlist := event.(*entity.Playlist)
		log.Println("[mixer]\t" + playlist.Name)
	}
}
