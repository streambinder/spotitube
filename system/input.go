package system

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// InputConfirm : ask for user confirmation over a given message
func InputConfirm(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [y/N]: ", message)
		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		response = string(strings.ToLower(strings.TrimSpace(response))[0])
		if response == "y" {
			return true
		}
		return false
	}
}

// InputString : ask for user input over a given message
func InputString(message string) string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println(message)
		response, err := reader.ReadString('\n')
		if err != nil {
			return ""
		}
		return response
	}
}
