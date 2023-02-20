package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmdSync(t *testing.T) {
	_, err := testExecute(cmdSync, "sync",
		"-l", "--library",
		"-p", "ID", "--playlist", "ID",
		"-a", "ID", "--album", "ID",
		"-t", "ID", "--track", "ID",
	)
	assert.Nil(t, err)
}

func TestCmdSyncWrong(t *testing.T) {
	_, err := testExecute(cmdSync, "sync", "--LIBRARY")
	assert.NotNil(t, err)
}
