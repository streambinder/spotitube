package cmd

import (
	"path/filepath"
	"strconv"

	"github.com/bogem/id3v2/v2"
	"github.com/spf13/cobra"
	"github.com/streambinder/spotitube/downloader"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/processor"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/util"
)

func init() {
	cmdRoot.AddCommand(cmdAttach())
}

func cmdAttach() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "attach",
		Short:        "Connect local track with its Spotify counterpart",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				path      = args[0]
				id        = args[1]
				rename, _ = cmd.Flags().GetBool("rename")
			)

			localTrack, err := id3.Open(path, id3v2.Options{Parse: false})
			if err != nil {
				return err
			}
			defer localTrack.Close()

			client, err := spotify.Authenticate(spotify.BrowserProcessor)
			if err != nil {
				return err
			}

			spotifyTrack, err := client.Track(id)
			if err != nil {
				return err
			}

			uslt, err := lyrics.Search(spotifyTrack)
			if err != nil {
				return err
			}

			artwork := make(chan []byte, 1)
			defer close(artwork)
			if err := downloader.Download(
				spotifyTrack.Artwork.URL, spotifyTrack.Path().Artwork(),
				processor.Artwork{}, artwork); err != nil {
				return err
			}

			localTrack.SetSpotifyID(spotifyTrack.ID)
			localTrack.SetTitle(spotifyTrack.Title)
			localTrack.SetArtist(spotifyTrack.Artists[0])
			localTrack.SetAlbum(spotifyTrack.Album)
			localTrack.SetArtworkURL(spotifyTrack.Artwork.URL)
			localTrack.SetAttachedPicture(<-artwork)
			localTrack.SetDuration(strconv.Itoa(spotifyTrack.Duration))
			localTrack.SetUnsynchronizedLyrics(spotifyTrack.Title, uslt)
			localTrack.SetTrackNumber(strconv.Itoa(spotifyTrack.Number))
			localTrack.SetYear(strconv.Itoa(spotifyTrack.Year))
			localTrack.SetUpstreamURL(spotifyTrack.UpstreamURL)

			if err := localTrack.Save(); err != nil {
				return err
			}

			if rename {
				return util.FileMoveOrCopy(path, filepath.Join(filepath.Dir(path), spotifyTrack.Path().Final()))
			}
			return nil
		},
	}
	cmd.Flags().BoolP("rename", "r", false, "Rename local track to comply with Spotify counterpart")
	return cmd
}
