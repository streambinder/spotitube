package cmd

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	test_exit_code := m.Run()
	if test_exit_code == 0 && testing.CoverMode() != "" && testing.Coverage() < 1 {
		fmt.Println("FAIL\tcoverage")
		test_exit_code = -1
	} else {
		fmt.Println("PASS\tcoverage")
	}
	os.Exit(test_exit_code)
}
