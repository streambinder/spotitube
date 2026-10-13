package cmd

import (
	"github.com/spf13/cobra"
	"github.com/streambinder/pango/spotify"
)

var (
	spotifyClient *spotify.Client
	cmdRoot       = &cobra.Command{
		Use:   "pango",
		Short: "Synchronize Spotify collections downloading from external providers",
	}
)

func Execute() error {
	return cmdRoot.Execute()
}
