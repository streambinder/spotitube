package cmd

import (
	"io"
	"log"
	"testing"
)

func BenchmarkRoot(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = testExecute()
	}
}

func testExecute(args ...string) error {
	log.SetOutput(io.Discard)
	cmdRoot.SetArgs(args)
	cmdRoot.SetOut(io.Discard)
	cmdRoot.SetErr(io.Discard)
	cmdRoot.SetOutput(io.Discard)
	return cmdRoot.Execute()
}

func TestExecute(t *testing.T) {
	cmdRoot.SetOut(io.Discard)
	cmdRoot.SetErr(io.Discard)
	cmdRoot.SetOutput(io.Discard)
	Execute()
}
