package cmd

import (
	"os/exec"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
)

func TestYouTubeDlDownload(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(*exec.Cmd) error { return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run")

	// testing
	assert.Nil(t, YouTubeDl("http://localhost", "fname.txt"))
}
