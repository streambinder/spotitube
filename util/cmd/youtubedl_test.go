package cmd

import (
	"os/exec"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestYouTubeDlDownload(t *testing.T) {
	// monkey patching
	patchexecCmdRun := gomonkey.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(*exec.Cmd) error { return nil })
	defer patchexecCmdRun.Reset()

	// testing
	assert.Nil(t, YouTubeDl("http://localhost", "fname.txt"))
}
