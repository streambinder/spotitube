package cmd

import (
	"github.com/spf13/cobra"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
)

var (
	spotifyClient *spotify.Client
	cmdRoot       = &cobra.Command{
		Use:   "spotitube",
		Short: "Synchronize Spotify collections downloading from external providers",
	}
)

func Execute() {
	sys.ErrSuppress(cmdRoot.Execute())
}
