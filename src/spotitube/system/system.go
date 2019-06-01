package system

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// PathsArrayFlag : struct containing all the informations about a parsed PathsArrayFlag input flag
type PathsArrayFlag struct {
	Paths []string
}

const (
	// Version : current version
	Version = 25
	// VersionRepository : repositoy container
	VersionRepository = "https://github.com/streambinder/spotitube"
	// VersionOrigin : API repository latest version URL
	VersionOrigin = "https://api.github.com/repos/streambinder/spotitube/releases/latest"
	// VersionURL : latest version for download
	VersionURL = VersionRepository + "/releases/latest"

	// ConcurrencyLimit : max concurrent jobs
	ConcurrencyLimit = 100

	// SongExtension : default downloaded songs extension
	SongExtension = ".mp3"
	// TCPCheckOrigin : default internet connection check origin
	TCPCheckOrigin = "github.com:443"
	// HTTPTimeout : default timeout for HTTP calls
	HTTPTimeout = 3 // second(s)

	// SystemLetterBytes : random string generator characters
	SystemLetterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// SystemLetterIdxBits : random string generator bits
	SystemLetterIdxBits = 6
	// SystemLetterIdxMask : random string generator mask
	SystemLetterIdxMask = 1<<SystemLetterIdxBits - 1
	// SystemLetterIdxMax : random string generator max
	SystemLetterIdxMax = 63 / SystemLetterIdxBits
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

// Mkdir : create directory dir if not already existing
func Mkdir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
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

// FileReadLines : open, read and return slice of file lines
func FileReadLines(path string) []string {
	var (
		lines     = make([]string, 0)
		file, err = os.Open(path)
	)

	if err != nil {
		return lines
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

// FileWriteLines : open and write slice of lines into file
func FileWriteLines(path string, lines []string) error {
	if err := os.Remove(path); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}
	return writer.Flush()
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

// LocalConfigPath : get local configuration and cache path
func LocalConfigPath() string {
	currentUser, _ := user.Current()
	return fmt.Sprintf("%s/.cache/spotitube", currentUser.HomeDir)
}

// DumpGob : serialize and dump to disk given object to give filePath path
func DumpGob(filePath string, object interface{}) error {
	file, err := os.Create(filePath)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(object)
	}
	file.Close()
	return err
}

// FetchGob : load previously dumped object from filePath to given object
func FetchGob(filePath string, object interface{}) error {
	file, err := os.Open(filePath)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(object)
	}
	file.Close()
	return err
}

// Asciify : transform eventually unicoded string to ASCII
func Asciify(dirty string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isNonspacingMark), norm.NFC)
	clean, _, _ := transform.String(t, dirty)
	return clean
}

func isNonspacingMark(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}
