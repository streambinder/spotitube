package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var cmdRoot = &cobra.Command{
	Use:   "spotitube",
	Short: "Synchronize Spotify collections downloading from external providers",
}

func Execute() {
	if cmdRoot.Execute() != nil {
		os.Exit(1)
	}
}

func init() {
	// cmdRoot.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
}
