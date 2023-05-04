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
	"github.com/streambinder/spotitube/entity/index"
	"github.com/streambinder/spotitube/entity/playlist"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/processor"
	"github.com/streambinder/spotitube/provider"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/util"
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
	indexData         = index.New()
	cmdSync           = &cobra.Command{
		Use:          "sync",
		Short:        "Synchronize collections",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				path, _             = cmd.Flags().GetString("path")
				playlistEncoding, _ = cmd.Flags().GetString("playlist-encoding")
				library, _          = cmd.Flags().GetBool("library")
				playlists, _        = cmd.Flags().GetStringArray("playlist")
				albums, _           = cmd.Flags().GetStringArray("album")
				tracks, _           = cmd.Flags().GetStringArray("track")
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
				routineMix(playlistEncoding),
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
	cmdSync.Flags().String("playlist-encoding", "m3u", "Target synchronization path")
	cmdSync.Flags().BoolP("library", "l", false, "Synchronize library (auto-enabled if no collection is supplied)")
	cmdSync.Flags().StringArrayP("playlist", "p", []string{}, "Synchronize playlist")
	cmdSync.Flags().StringArrayP("album", "a", []string{}, "Synchronize album")
	cmdSync.Flags().StringArrayP("track", "t", []string{}, "Synchronize track")
}

// indexer scans a possible local music library
// to be considered as already synchronized
func routineIndex(ctx context.Context, ch chan error) {
	// remember to signal fetcher
	defer close(routineSemaphores[routineTypeIndex])

	if err := indexData.Build("."); err != nil {
		log.Println(util.Excerpt("[indexer]", true), err)
		routineSemaphores[routineTypeIndex] <- false
		ch <- err
		return
	}
	log.Println(util.Excerpt("[indexer]", true), "indexed", len(indexData))

	// once indexed, sidgnal fetcher
	routineSemaphores[routineTypeIndex] <- true
}

func routineAuth(ctx context.Context, ch chan error) {
	// remember to close auth semaphore
	defer close(routineSemaphores[routineTypeAuth])

	var err error
	spotifyClient, err = spotify.Authenticate()
	if err != nil {
		log.Println(util.Excerpt("[auth]", true), err)
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
		if !<-routineSemaphores[routineTypeIndex] {
			return
		}
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

	for event := range routineQueues[routineTypeDecide] {
		track := event.(*entity.Track)

		if status, ok := indexData[track.ID]; !ok {
			log.Println(util.Excerpt("[decider]", true), util.Excerpt(track.Title, true), "sync")
			indexData[track.ID] = index.Online
		} else if status == index.Online {
			log.Println(util.Excerpt("[decider]", true), util.Excerpt(track.Title, true), "skip (dup)")
			continue
		} else if status == index.Offline {
			// log.Println(util.Excerpt("[decider]", true), util.Excerpt(track.Title, true), "skip (offline)")
			continue
		}

		matches, err := provider.Search(track)
		if err != nil {
			ch <- err
			return
		}

		if len(matches) == 0 {
			log.Println(util.Excerpt("[decider]", true), util.Excerpt(track.Title, true), "no result")
			continue
		}

		track.UpstreamURL = matches[0].URL
		routineQueues[routineTypeCollect] <- track
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
		log.Println(util.Excerpt("[collect]", true), util.Excerpt(track.Title, true))
		if err := nursery.RunConcurrently(
			routineCollectAsset(track),
			routineCollectLyrics(track),
			routineCollectArtwork(track),
		); err != nil {
			log.Println(util.Excerpt("[collect]", true), util.Excerpt(track.Title, true), err)
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
		log.Println(util.Excerpt("[retriever]", true), util.Excerpt(track.Title, true), track.UpstreamURL)
		if err := downloader.Download(track.UpstreamURL, track.Path().Download(), nil); err != nil {
			log.Println(util.Excerpt("[retriever]", true), util.Excerpt(track.Title, true), err)
			ch <- err
			return
		}
	}
}

// composer pulls lyrics to be inserted
// in the fetched blob
func routineCollectLyrics(track *entity.Track) func(context.Context, chan error) {
	return func(ctx context.Context, ch chan error) {
		log.Println(util.Excerpt("[composer]", true), util.Excerpt(track.Title, true))
		lyrics, err := lyrics.Search(track)
		if err != nil {
			log.Println(util.Excerpt("[composer]", true), util.Excerpt(track.Title, true), err)
			ch <- err
			return
		}
		track.Lyrics = lyrics
		log.Println(util.Excerpt("[composer]", true), util.Excerpt(track.Title, true), util.Excerpt(lyrics))
	}
}

// painter pulls image blobs to be inserted
// as artworks in the fetched blob
func routineCollectArtwork(track *entity.Track) func(context.Context, chan error) {
	return func(ctx context.Context, ch chan error) {
		artwork := make(chan []byte, 1)
		defer close(artwork)

		log.Println(util.Excerpt("[painter]", true), util.Excerpt(track.Title, true), track.Artwork.URL)
		if err := downloader.Download(track.Artwork.URL, track.Path().Artwork(), processor.Artwork{}, artwork); err != nil {
			log.Println(util.Excerpt("[painter]", true), util.Excerpt(track.Title, true), err)
			ch <- err
			return
		}

		track.Artwork.Data = <-artwork
		log.Println(util.Excerpt("[painter]", true), util.Excerpt(track.Title, true), len(track.Artwork.Data))
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
		log.Println(util.Excerpt("[postproc]", true), util.Excerpt(track.Title, true))
		if err := processor.Do(track); err != nil {
			log.Println(util.Excerpt("[postproc]", true), util.Excerpt(track.Title, true), err)
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
		log.Println(util.Excerpt("[installer]", true), util.Excerpt(track.Title, true))
		if err := util.FileMoveOrCopy(track.Path().Download(), track.Path().Final()); err != nil {
			log.Println(util.Excerpt("[installer]", true), util.Excerpt(track.Title, true), err)
			ch <- err
			return
		}
		indexData[track.ID] = index.Installed
	}
}

// mixer wraps playlists to their final destination
func routineMix(encoding string) func(context.Context, chan error) {
	return func(ctx context.Context, ch chan error) {
		// block until installation is done
		<-routineSemaphores[routineTypeInstall]

		for event := range routineQueues[routineTypeMix] {
			playlist := event.(*playlist.Playlist)
			log.Println(util.Excerpt("[mixer]", true), util.Excerpt(playlist.Name, true))
			encoder, err := playlist.Encoder(encoding)
			if err != nil {
				log.Println(util.Excerpt("[mixer]", true), util.Excerpt(playlist.Name, true), err)
				ch <- err
				return
			}

			for _, track := range playlist.Tracks {
				if trackStatus, ok := indexData[track.ID]; !ok || (trackStatus != index.Installed && trackStatus != index.Offline) {
					continue
				}

				if err := encoder.Add(track); err != nil {
					log.Println(util.Excerpt("[mixer]", true), util.Excerpt(playlist.Name, true), err)
					ch <- err
					return

				}
			}

			if err := encoder.Close(); err != nil {
				log.Println(util.Excerpt("[mixer]", true), util.Excerpt(playlist.Name, true), err)
				ch <- err
				return
			}
		}
	}
}
