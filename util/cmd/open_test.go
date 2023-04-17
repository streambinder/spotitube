package cmd

import (
	"os/exec"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestOpen(t *testing.T) {
	// monkey patching
	patchexecCmdStart := gomonkey.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "Start", func(*exec.Cmd) error { return nil })
	defer patchexecCmdStart.Reset()

	// testing
	assert.Nil(t, Open("https://davidepucci.it", "linux"))
	assert.Nil(t, Open("https://davidepucci.it", "darwin"))
	assert.Nil(t, Open("https://davidepucci.it", "windows"))
	assert.EqualError(t, Open("https://davidepucci.it", "unknown"), "unsupported platform")
}
