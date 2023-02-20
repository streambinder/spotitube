package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// func TestMain(m *testing.M) {
// 	returnCode := m.Run()
// 	if returnCode == 0 && testing.CoverMode() != "" {
// 		if testing.Coverage() < 0.9 {
// 			fmt.Printf("Coverage not reached: %.1f", testing.Coverage())
// 			os.Exit(-1)
// 		}
// 	}
// }

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

func TestExecuteOk(t *testing.T) {
	Execute()
}
