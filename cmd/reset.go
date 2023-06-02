package cmd

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	"github.com/streambinder/spotitube/spotify"
)

func init() {
	cmdRoot.AddCommand(cmdReset())
}

func cmdReset() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "reset",
		Short:        "Clear cached objects",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			session, _ := cmd.Flags().GetBool("session")
			cachePath, err := xdg.CacheFile("spotitube")
			if err != nil {
				return err
			}

			return filepath.WalkDir(cachePath, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if cachePath == path || (entry.Name() == spotify.TokenBasename && !session) {
					return nil
				}

				return os.RemoveAll(path)
			})
		},
	}
	cmd.Flags().BoolP("session", "s", false, "Logout from active sessions")
	return cmd
}
