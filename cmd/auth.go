package cmd

import (
	"errors"
	"io/fs"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
)

var printProcessor = func(url string) error {
	log.Println("Authenticate at:", url)
	return nil
}

func init() {
	cmdRoot.AddCommand(cmdAuth())
}

func cmdAuth() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "auth",
		Short:        "Establish a Spotify session for future uses",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var (
				remote    = sys.ErrWrap(false)(cmd.Flags().GetBool("remote"))
				logout    = sys.ErrWrap(false)(cmd.Flags().GetBool("logout"))
				callback  = "127.0.0.1"
				processor = spotify.BrowserProcessor
			)
			if remote {
				log.Println("In order for remote authentication to work, set DNS/hosts entry to make `spotitube.local` resolve to the Spotitube server")
				callback = "spotitube.local"
				processor = printProcessor
			}

			if logout {
				if err := os.Remove(sys.CacheFile(spotify.TokenBasename)); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}
			}

			return sys.ErrOnly(spotify.Authenticate(processor, callback))
		},
	}
	cmd.Flags().BoolP("remote", "r", false, "Spotitube server is remote")
	cmd.Flags().BoolP("logout", "l", false, "Logout before starting authentication process")
	return cmd
}
