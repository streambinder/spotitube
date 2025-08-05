package cmd

import (
	"io"
	"log"
	"testing"

	"github.com/spf13/cobra"
	"github.com/streambinder/spotitube/sys"
)

func BenchmarkRoot(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sys.ErrSuppress(testExecute(cmdRoot))
	}
}

func testExecute(cmd *cobra.Command, args ...string) error {
	cmdTest := &cobra.Command{}
	cmdTest.AddCommand(cmd)
	log.SetOutput(io.Discard)
	cmdTest.SetArgs(append([]string{cmd.Use}, args...))
	cmdTest.SetOut(io.Discard)
	cmdTest.SetErr(io.Discard)
	return cmdTest.Execute()
}

func TestExecute(_ *testing.T) {
	cmdRoot.SetOut(io.Discard)
	cmdRoot.SetErr(io.Discard)
	Execute()
}
