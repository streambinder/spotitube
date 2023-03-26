package cmd

import (
	"os/exec"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
)

func TestOpen(t *testing.T) {
	// monkey patching
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Start", func(_ *exec.Cmd) error { return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Start")

	// testing
	assert.Nil(t, Open("https://davidepucci.it", "linux"))
	assert.Nil(t, Open("https://davidepucci.it", "darwin"))
	assert.Nil(t, Open("https://davidepucci.it", "windows"))
	assert.EqualError(t, Open("https://davidepucci.it", "unknown"), "unsupported platform")
}
