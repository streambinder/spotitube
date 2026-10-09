package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
	"go.uber.org/goleak"
)

// Command tests seed and delete the real Spotify session file: the path
// is frozen in a package variable of the spotify package at startup, so
// it cannot be redirected to a temp directory from here. Any pre-existing
// session is moved aside for the whole run and restored at the next one:
// goleak.VerifyTestMain exits the process, so the restore cannot happen
// at the end of this run.
func TestMain(m *testing.M) {
	session := sys.CacheFile(spotify.TokenBasename)
	backup := session + ".test-backup"
	if _, err := os.Stat(backup); err == nil {
		moveAside(backup, session)
	}
	if _, err := os.Stat(session); err == nil {
		moveAside(session, backup)
	}

	goleak.VerifyTestMain(m)
}

// moveAside renames a session file, aborting the whole test run if the
// rename fails: running against a live session would corrupt it.
func moveAside(from, to string) {
	if err := os.Rename(from, to); err != nil {
		fmt.Fprintln(os.Stderr, "spotitube cmd tests: cannot move session file:", err)
		os.Exit(1)
	}
}
