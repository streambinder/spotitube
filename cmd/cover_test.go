package cmd

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if m.Run() == 0 &&
		testing.CoverMode() != "" &&
		testing.Coverage() != 1.0 {
		fmt.Printf("FAIL\tcoverage %.1f\n", testing.Coverage())
		os.Exit(-1)
	}
}
