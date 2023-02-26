package cmd

import (
	"bytes"
	"testing"
)

func testExecute(args ...string) (stdout, stderr string, err error) {
	var (
		stdoutBuffer = new(bytes.Buffer)
		stderrBuffer = new(bytes.Buffer)
	)
	cmdRoot.SetArgs(args)
	cmdRoot.SetErr(stderrBuffer)
	cmdRoot.SetOutput(stdoutBuffer)
	err = cmdRoot.Execute()
	stderr = stderrBuffer.String()
	stdout = stdoutBuffer.String()
	return
}

func TestExecute(t *testing.T) {
	Execute()
}
