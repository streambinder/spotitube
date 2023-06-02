package cmd

import (
	"context"
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
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
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
			tracks, _ := cmd.Flags().GetStringArray("track")
			client, err := spotify.Authenticate()
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
					if len(tracks) > 0 {
						for _, id := range tracks {
							if _, err := client.Track(id, providerChannel, lyricsChannel); err != nil {
								ch <- err
								return
							}
						}
					} else {
						if err := client.Library(providerChannel, lyricsChannel); err != nil {
							ch <- err
							return
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
	cmd.Flags().StringArrayP("track", "t", []string{}, "Lookup given tracks only")
	return cmd
}
