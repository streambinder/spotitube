package cmd

import (
	"errors"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
)

func init() {
	cmdRoot.AddCommand(cmdAuth())
}

func cmdAuth() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "auth",
		Short:        "Establish a Spotify session for future uses",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if sys.ErrWrap(false)(cmd.Flags().GetBool("logout")) {
				if err := os.Remove(sys.CacheFile(spotify.TokenBasename)); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}
			}

			return sys.ErrOnly(spotify.Authenticate(spotify.BrowserProcessor))
		},
	}
	cmd.Flags().BoolP("logout", "l", false, "Logout before starting authentication process")
	return cmd
}
