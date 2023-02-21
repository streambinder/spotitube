package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	test_exit_code := m.Run()
	if test_exit_code == 0 && testing.CoverMode() != "" && testing.Coverage() < 1 {
		fmt.Println("FAIL\tcoverage")
		test_exit_code = -1
	}
	os.Exit(test_exit_code)
}

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
	assert.Nil(t, Execute())
}
