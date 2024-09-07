package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/arunsworld/nursery"
	"github.com/spf13/cobra"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/provider"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/util"
)

const (
	colorReset        = "\033[0m"
	colorRed          = "\033[31m"
	defaultRandomSize = 5
)

func init() {
	cmdRoot.AddCommand(cmdLookup())
}

func cmdLookup() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "lookup",
		Short:        "Utility to lookup for tracks in order to investigate general querying behaviour",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			library := util.ErrWrap(false)(cmd.Flags().GetBool("library"))
			random := util.ErrWrap(false)(cmd.Flags().GetBool("random"))
			randomSize := util.ErrWrap(defaultRandomSize)(cmd.Flags().GetInt("random-size"))
			libraryLimit := util.ErrWrap(0)(cmd.Flags().GetInt("library-limit"))
			if !library && !random && len(args) == 0 {
				return errors.New("no track has been issued")
			}

			var authErr error
			spotifyClient, authErr = spotify.Authenticate(spotify.BrowserProcessor)
			if authErr != nil {
				return authErr
			}

			var (
				providerChannel = make(chan interface{}, 1)
				lyricsChannel   = make(chan interface{}, 1)
			)
			return nursery.RunConcurrently(
				routineLookupFetch(random, library, randomSize, libraryLimit, args, providerChannel, lyricsChannel),
				routineLookupProvider(providerChannel),
				routineLookupLyrics(lyricsChannel),
			)
		},
	}
	cmd.Flags().BoolP("library", "l", false, "Lookup personal library tracks")
	cmd.Flags().BoolP("random", "r", false, "Lookup random tracks")
	cmd.Flags().Int("random-size", defaultRandomSize, "Number of random tracks to load")
	cmd.Flags().Int("library-limit", 0, "Number of tracks to fetch from library (unlimited if 0)")
	return cmd
}

func routineLookupFetch(random, library bool, randomSize, libraryLimit int, ids []string, providerChannel, lyricsChannel chan interface{}) func(context.Context, chan error) {
	return func(_ context.Context, ch chan error) {
		defer close(providerChannel)
		defer close(lyricsChannel)

		switch {
		case random:
			if err := spotifyClient.Random(spotify.TypeTrack, randomSize, providerChannel, lyricsChannel); err != nil {
				ch <- err
				return
			}
		case library:
			if err := spotifyClient.Library(libraryLimit, providerChannel, lyricsChannel); err != nil {
				ch <- err
				return
			}
		default:
			for _, id := range ids {
				if _, err := spotifyClient.Track(id, providerChannel, lyricsChannel); err != nil {
					ch <- err
					return
				}
			}
		}
	}
}

func routineLookupProvider(providerChannel chan interface{}) func(context.Context, chan error) {
	return func(context.Context, chan error) {
		prefix := "[P]"
		for event := range providerChannel {
			track := event.(*entity.Track)
			matches, err := provider.Search(track)
			switch {
			case err != nil:
				fmt.Println(colorRed+prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), err, colorReset)
			case len(matches) == 0:
				fmt.Println(colorRed+prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), "no result", colorReset)
			default:
				fmt.Println(prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), matches[0].URL, matches[0].Score)
			}
		}
	}
}

func routineLookupLyrics(lyricsChannel chan interface{}) func(context.Context, chan error) {
	return func(context.Context, chan error) {
		prefix := "[L]"
		for event := range lyricsChannel {
			track := event.(*entity.Track)
			lyrics, err := lyrics.Search(track)
			switch {
			case err != nil:
				fmt.Println(colorRed+prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), err, colorReset)
			case len(lyrics) == 0:
				fmt.Println(colorRed+prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), "no result", colorReset)
			default:
				fmt.Println(prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), util.Excerpt(lyrics, 80))
			}
		}
	}
}
