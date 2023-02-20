package cmd

import (
	"bytes"
	"testing"

	"github.com/matryer/is"
	"github.com/spf13/cobra"
)

func testExecute(subcmd *cobra.Command, args ...string) (output string, err error) {
	var (
		outputBuffer = new(bytes.Buffer)
		command      = &cobra.Command{}
	)
	command.SetOutput(outputBuffer)
	command.SetErr(outputBuffer)
	command.SetArgs(args)
	command.AddCommand(subcmd)
	err = command.Execute()
	output = outputBuffer.String()
	return
}

func TestCmdRootHelp(t *testing.T) {
	cmdRoot.SetArgs([]string{"--help"})
	is.New(t).NoErr(cmdRoot.Execute())
}

func TestCmdRootHelpP(t *testing.T) {
	cmdRoot.SetArgs([]string{"-h"})
	is.New(t).NoErr(cmdRoot.Execute())
}

func TestCmdRootHelpS(t *testing.T) {
	cmdRoot.SetArgs([]string{"help"})
	is.New(t).NoErr(cmdRoot.Execute())
}

func TestCmdRootAutocomplete(t *testing.T) {
	cmdRoot.SetArgs([]string{"completion", "bash"})
	is.New(t).NoErr(cmdRoot.Execute())
}
