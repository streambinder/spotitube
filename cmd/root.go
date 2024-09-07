package cmd

import (
	"github.com/spf13/cobra"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/util"
)

var (
	spotifyClient *spotify.Client
	cmdRoot       = &cobra.Command{
		Use:   "spotitube",
		Short: "Synchronize Spotify collections downloading from external providers",
	}
)

func Execute() {
	util.ErrSuppress(cmdRoot.Execute())
}
