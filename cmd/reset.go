package cmd

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
)

func init() {
	cmdRoot.AddCommand(cmdReset())
}

func cmdReset() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "reset",
		Short:        "Clear cached objects",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var (
				session        = sys.ErrWrap(false)(cmd.Flags().GetBool("session"))
				cacheDirectory = sys.CacheDirectory()
			)
			return filepath.WalkDir(cacheDirectory, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if cacheDirectory == path || (entry.Name() == spotify.TokenBasename && !session) {
					return nil
				}

				return os.RemoveAll(path)
			})
		},
	}
	cmd.Flags().BoolP("session", "s", false, "Logout from active sessions")
	return cmd
}
