package cmd

import (
	"io"
	"log"
	"testing"

	"github.com/spf13/cobra"
)

func BenchmarkRoot(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = testExecute(cmdRoot)
	}
}

func testExecute(cmd *cobra.Command, args ...string) error {
	cmdTest := &cobra.Command{}
	cmdTest.AddCommand(cmd)
	log.SetOutput(io.Discard)
	cmdTest.SetArgs(append([]string{cmd.Use}, args...))
	cmdTest.SetOut(io.Discard)
	cmdTest.SetErr(io.Discard)
	cmdTest.SetOutput(io.Discard)
	return cmdTest.Execute()
}

func TestExecute(t *testing.T) {
	cmdRoot.SetOut(io.Discard)
	cmdRoot.SetErr(io.Discard)
	cmdRoot.SetOutput(io.Discard)
	Execute()
}
