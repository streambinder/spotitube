package cmd

import (
	"context"
	"log"

	"github.com/arunsworld/nursery"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streambinder/spotitube/spotify"
)

const (
	index int = iota
	fetch
	download
	paint
	compose
	process
	install
)

var (
	queues  map[int](chan spotify.ID)
	cmdSync = &cobra.Command{
		Use:   "sync",
		Short: "Synchronize collections",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return nursery.RunConcurrently(
				indexer,
				fetcher,
				downloader,
				painter,
				composer,
				processor,
				installer,
			)
		},
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			queues = map[int](chan spotify.ID){
				fetch:    make(chan spotify.ID),
				download: make(chan spotify.ID),
				paint:    make(chan spotify.ID),
				compose:  make(chan spotify.ID),
				process:  make(chan spotify.ID),
				install:  make(chan spotify.ID),
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
	defer close(queues[fetch])

	log.Println("[indexer]\tindexing")
	log.Println("[indexer]\tindexed")
}

// fetcher pulls data from the upstream
// provider, i.e. Spotify
func fetcher(context.Context, chan error) {
	// remember to stop passing data to downloader
	defer close(queues[download])
	// block until indexig is done
	<-queues[fetch]

	log.Println("[fetcher]\tfetching")
	queues[download] <- spotify.ID("6rqhFgbbKwnb9MLmUQDhG6")
	log.Println("[fetcher]\tfetched")
}

// downloader pulls a track blob corresponding
// to the (meta)data fetched from upstream
func downloader(context.Context, chan error) {
	// remember to stop passing data to painter
	defer close(queues[paint])

	for track := range queues[download] {
		log.Println("[download]\t" + track)
		queues[paint] <- track
	}
}

// painter pulls image blobs to be inserted
// as artworks in the fetched blob
func painter(context.Context, chan error) {
	// remember to stop passing data to composer
	defer close(queues[compose])

	for track := range queues[paint] {
		log.Println("[painter]\t" + track)
		queues[compose] <- track
	}
}

// composer pulls lyrics to be inserted
// in the fetched blob
func composer(context.Context, chan error) {
	// remember to stop passing data to processor
	defer close(queues[process])

	for track := range queues[compose] {
		log.Println("[composer]\t" + track)
		queues[process] <- track
	}
}

// processor applies some further enhancements
// e.g. combining the downloaded artwork/lyrics
// into the blob
func processor(context.Context, chan error) {
	// remember to stop passing data to installer
	defer close(queues[install])

	for track := range queues[process] {
		log.Println("[processor]\t" + track)
		queues[install] <- track
	}
}

// installer move the blob to its final destination
func installer(context.Context, chan error) {
	for track := range queues[install] {
		log.Println("[installer]\t" + track)
	}
}
