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
			library, _ := cmd.Flags().GetBool("library")
			random, _ := cmd.Flags().GetBool("random")
			randomSize, _ := cmd.Flags().GetInt("random-size")
			libraryLimit, _ := cmd.Flags().GetInt("library-limit")
			if !library && !random && len(args) == 0 {
				return errors.New("no track has been issued")
			}

			client, err := spotify.Authenticate(spotify.BrowserProcessor)
			if err != nil {
				return err
			}

			var (
				providerChannel = make(chan interface{}, 1)
				lyricsChannel   = make(chan interface{}, 1)
			)
			return nursery.RunConcurrently(
				func(ctx context.Context, ch chan error) {
					defer close(providerChannel)
					defer close(lyricsChannel)
					if random {
						if err := client.Random(spotify.TypeTrack, randomSize, providerChannel, lyricsChannel); err != nil {
							ch <- err
							return
						}
					} else if library {
						if err := client.Library(libraryLimit, providerChannel, lyricsChannel); err != nil {
							ch <- err
							return
						}
					} else {
						for _, id := range args {
							if _, err := client.Track(id, providerChannel, lyricsChannel); err != nil {
								ch <- err
								return
							}
						}
					}
				},
				func(ctx context.Context, ch chan error) {
					prefix := "[P]"
					for event := range providerChannel {
						track := event.(*entity.Track)
						matches, err := provider.Search(track)
						if err != nil {
							fmt.Println(colorRed+prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), err, colorReset)
						} else if len(matches) == 0 {
							fmt.Println(colorRed+prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), "no result", colorReset)
						} else {
							fmt.Println(prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), matches[0].URL, matches[0].Score)
						}
					}
				},
				func(ctx context.Context, ch chan error) {
					prefix := "[L]"
					for event := range lyricsChannel {
						track := event.(*entity.Track)
						lyrics, err := lyrics.Search(track)
						if err != nil {
							fmt.Println(colorRed+prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), err, colorReset)
						} else if len(lyrics) == 0 {
							fmt.Println(colorRed+prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), "no result", colorReset)
						} else {
							fmt.Println(prefix, track.ID, util.Pad(track.Artists[0]), util.Pad(track.Title), util.Excerpt(lyrics, 80))
						}
					}
				},
			)
		},
	}
	cmd.Flags().BoolP("library", "l", false, "Lookup personal library tracks")
	cmd.Flags().BoolP("random", "r", false, "Lookup random tracks")
	cmd.Flags().Int("random-size", defaultRandomSize, "Number of random tracks to load")
	cmd.Flags().Int("library-limit", 0, "Number of tracks to fetch from library (unlimited if 0)")
	return cmd
}
