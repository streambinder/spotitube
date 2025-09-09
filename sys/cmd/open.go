package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
)

func Open(url string, oses ...string) (err error) {
	os := runtime.GOOS
	if len(oses) > 0 {
		os = oses[0]
	}

	switch os {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}
