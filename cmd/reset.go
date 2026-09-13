package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
)

// indirection for (*os.Root).RemoveAll — the method is aggressively inlined
// by the compiler, making it unpatchable by mockey without -gcflags
var rootRemoveAll = (*os.Root).RemoveAll

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
			root, err := os.OpenRoot(cacheDirectory)
			if err != nil {
				return err
			}
			defer root.Close()
			return filepath.WalkDir(cacheDirectory, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if cacheDirectory == path || (entry.Name() == spotify.TokenBasename && !session) {
					return nil
				}

				rel := strings.TrimPrefix(path, cacheDirectory+string(filepath.Separator))
				if err := rootRemoveAll(root, rel); err != nil {
					return err
				}
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolP("session", "s", false, "Logout from active sessions")
	return cmd
}
