package main

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMainHelp(_ *testing.T) {
	// a successful command execution returns from main without exiting
	os.Args = []string{"spotitube", "--help"}
	main()
}

func TestMainFailure(t *testing.T) {
	if os.Getenv("SPOTITUBE_MAIN_HELPER") == "1" {
		// helper process: an unknown command makes Execute fail and
		// main exit with a non-zero code
		os.Args = []string{"spotitube", "no-such-command"}
		main()
		return
	}

	executable, err := os.Executable()
	assert.Nil(t, err)
	cmd := exec.Command(executable, "-test.run", "^TestMainFailure$")
	cmd.Env = append(os.Environ(), "SPOTITUBE_MAIN_HELPER=1")
	err = cmd.Run()
	var exitErr *exec.ExitError
	assert.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 1, exitErr.ExitCode())
}
