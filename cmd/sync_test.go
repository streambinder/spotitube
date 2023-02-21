package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmdSync(t *testing.T) {
	_, _, err := testExecute("sync",
		"-l", "--library",
		"-p", "ID", "--playlist", "ID",
		"-a", "ID", "--album", "ID",
		"-t", "ID", "--track", "ID",
	)
	assert.Nil(t, err)
}

func TestCmdSyncLibraryDynamicallySet(t *testing.T) {
	_, _, err := testExecute("sync")
	assert.Nil(t, err)
	library, err := cmdSync.Flags().GetBool("library")
	assert.Nil(t, err)
	assert.True(t, library)
}
