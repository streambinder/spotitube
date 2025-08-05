package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/streambinder/id3v2-sylt"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/sys"
)

const fallback = "<unset>"

func init() {
	cmdRoot.AddCommand(cmdShow())
}

func cmdShow() *cobra.Command {
	return &cobra.Command{
		Use:          "show",
		Short:        "Show local tracks data",
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			bold := color.New(color.Bold)
			for i, path := range args {
				if err := func() error {
					tag, err := id3.Open(path, id3v2.Options{Parse: true})
					if err != nil {
						return err
					}
					defer tag.Close()

					table := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.AlignRight)
					fmt.Fprintln(table, "Path\t", bold.Sprint(path))
					fmt.Fprintln(table, "Spotify ID\t", sys.Fallback(tag.SpotifyID(), fallback))
					fmt.Fprintln(table, "Title\t", sys.Fallback(tag.Title(), fallback))
					fmt.Fprintln(table, "Artist\t", sys.Fallback(tag.Artist(), fallback))
					fmt.Fprintln(table, "Album\t", sys.Fallback(tag.Album(), fallback))
					fmt.Fprintln(table, "Year\t", sys.Fallback(tag.Year(), fallback))
					fmt.Fprintln(table, "Track number\t", sys.Fallback(tag.TrackNumber(), fallback))
					fmt.Fprintln(table, "Artwork URL\t", sys.Fallback(tag.ArtworkURL(), fallback))
					fmt.Fprintln(table, "Duration\t", sys.Fallback(fmt.Sprintf("%ss", tag.Duration()), fallback))
					fmt.Fprintln(table, "Upstream URL\t", sys.Fallback(tag.UpstreamURL(), fallback))
					fmt.Fprintln(table, "Lyrics\t", sys.Fallback(sys.Excerpt(sys.FirstLine(tag.UnsynchronizedLyrics()), 64), fallback))
					fmt.Fprintln(table, "Artwork\t", func(mimeType string, data []byte) string {
						if len(data) > 0 {
							return fmt.Sprintf("%s (%s)", mimeType, sys.HumanizeBytes(len(data)))
						}
						return fallback
					}(tag.AttachedPicture()))
					if len(args) > 1 && i < len(args)-1 {
						fmt.Fprintln(table)
					}
					return table.Flush()
				}(); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
