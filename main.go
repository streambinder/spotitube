package main

import (
	"context"
	"log"
	"time"

	"github.com/arunsworld/nursery"
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

type track struct {
	title string
}

var queues map[int](chan track) = map[int](chan track){
	fetch:    make(chan track),
	download: make(chan track),
	paint:    make(chan track),
	compose:  make(chan track),
	process:  make(chan track),
	install:  make(chan track),
}

func init() {
}

func main() {
	if err := nursery.RunConcurrently(
		indexer,
		fetcher,
		downloader,
		painter,
		composer,
		processor,
		installer,
	); err != nil {
		log.Fatalln(err)
	}
}

// indexer scans a possible local music library
// to be considered as already synchronized
func indexer(context.Context, chan error) {
	// remember to signal fetcher
	defer close(queues[fetch])

	log.Println("[indexer]\tindexing")
	time.Sleep(2 * time.Second)
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
	queues[download] <- track{"title1"}
	time.Sleep(2 * time.Second)
	queues[download] <- track{"title2"}
	time.Sleep(1 * time.Second)
	log.Println("[fetcher]\tfetched")
}

// downloader pulls a track blob corresponding
// to the (meta)data fetched from upstream
func downloader(context.Context, chan error) {
	// remember to stop passing data to painter
	defer close(queues[paint])

	for track := range queues[download] {
		log.Println("[download]\t" + track.title)
		time.Sleep(5 * time.Second)
		queues[paint] <- track
	}
}

// painter pulls image blobs to be inserted
// as artworks in the fetched blob
func painter(context.Context, chan error) {
	// remember to stop passing data to composer
	defer close(queues[compose])

	for track := range queues[paint] {
		log.Println("[painter]\t" + track.title)
		time.Sleep(2 * time.Second)
		queues[compose] <- track
	}
}

// composer pulls lyrics to be inserted
// in the fetched blob
func composer(context.Context, chan error) {
	// remember to stop passing data to processor
	defer close(queues[process])

	for track := range queues[compose] {
		log.Println("[composer]\t" + track.title)
		time.Sleep(1 * time.Second)
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
		log.Println("[processor]\t" + track.title)
		queues[install] <- track
	}
}

// installer move the blob to its final destination
func installer(context.Context, chan error) {
	for track := range queues[install] {
		log.Println("[installer]\t" + track.title)
	}
}
