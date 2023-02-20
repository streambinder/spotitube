package cmd

import (
	"testing"

	"github.com/matryer/is"
)

func TestCmdSync(t *testing.T) {
	_, err := testExecute(cmdSync, "sync",
		"-l", "--library",
		"-p", "ID", "--playlist", "ID",
		"-a", "ID", "--album", "ID",
		"-t", "ID", "--track", "ID",
	)
	is.New(t).NoErr(err)
}

func TestCmdSyncWrong(t *testing.T) {
	_, err := testExecute(cmdSync, "sync", "--LIBRARY")
	is.New(t).True(err != nil)
}
