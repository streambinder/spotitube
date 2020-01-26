package system

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// InputConfirm asks for user confirmation over a given message
func InputConfirm(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [y/N]: ", message)
		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		response = string(strings.ToLower(strings.TrimSpace(response)))
		if len(response) > 0 && response[0] == 'y' {
			return true
		}
		return false
	}
}

// InputString asks for user input over a given message
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
