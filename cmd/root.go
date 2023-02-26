package cmd

import (
	"github.com/spf13/cobra"
)

var cmdRoot = &cobra.Command{
	Use:   "spotitube",
	Short: "Synchronize Spotify collections downloading from external providers",
}

func Execute() {
	_ = cmdRoot.Execute()
}
