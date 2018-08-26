package system

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// String : string representation for PathsArrayFlag object
func (flag *PathsArrayFlag) String() string {
	return fmt.Sprint(flag.Paths)
}

// Set : set value of a PathsArrayFlag object
func (flag *PathsArrayFlag) Set(value string) error {
	paths := strings.Split(value, ";")
	for _, path := range paths {
		pathAbs, pathErr := filepath.Abs(path)
		if pathErr != nil {
			return pathErr
		}
		flag.Paths = append(flag.Paths, pathAbs)
	}
	return nil
}

// Dir : return True if input string path is a directory
func Dir(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	fileStat, err := file.Stat()
	if err != nil {
		return false
	}
	return fileStat.IsDir()
}

// MakeRange : return a range array between input int(s) min and max
func MakeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

// RandString : return a (input int) n-long random string
func RandString(n int) string {
	var src = rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), SystemLetterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), SystemLetterIdxMax
		}
		if idx := int(cache & SystemLetterIdxMask); idx < len(SystemLetterBytes) {
			b[i] = SystemLetterBytes[idx]
			i--
		}
		cache >>= SystemLetterIdxBits
		remain--
	}

	return string(b)
}

// FileExists : return True if input string path points to a valid file
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// FileTouch : create file in input string path
func FileTouch(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

// FileCopy : copy file from input string pathFrom to input string pathTo
func FileCopy(pathFrom string, pathTo string) error {
	pathFromOpen, err := os.Open(pathFrom)
	if err != nil {
		return err
	}
	defer pathFromOpen.Close()

	pathToOpen, err := os.Create(pathTo)
	if err != nil {
		return err
	}

	if _, err := io.Copy(pathToOpen, pathFromOpen); err != nil {
		pathToOpen.Close()
		return err
	}

	return pathToOpen.Close()
}

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
