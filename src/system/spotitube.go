package system

import (
	"fmt"
	"os/user"
)

// LocalConfigPath : get local configuration and cache path
func LocalConfigPath() string {
	currentUser, _ := user.Current()
	return fmt.Sprintf("%s/.cache/spotitube", currentUser.HomeDir)
}
