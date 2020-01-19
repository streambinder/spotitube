package system

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Dir returns True if given path is a directory
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

// Mkdir creates directory dir if not already existing
func Mkdir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

// PrettyPath replaces home directory sequence with a tilde
func PrettyPath(dir string) string {
	dir, _ = filepath.Abs(dir)

	if u, err := user.Current(); err == nil {
		dir = strings.Replace(dir, u.HomeDir, "~", -1)
	}

	return dir
}

// FileExists returns True if given path points to a valid file
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// FileTouch creates file to given path
func FileTouch(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

// FileCopy copies file from given pathFrom to given pathTo
func FileCopy(pathFrom, pathTo string) error {
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

// FileMove moves file from given pathFrom to given pathTo
func FileMove(pathFrom string, pathTo string) error {
	if err := FileCopy(pathFrom, pathTo); err != nil {
		return err
	}

	return os.Remove(pathFrom)
}

// FileWildcardDelete deletes files from an array of wildcard strings
func FileWildcardDelete(path string, wildcards ...string) int {
	var deletions int

	for _, wildcard := range wildcards {
		files, err := filepath.Glob(wildcard)
		if err != nil {
			continue
		}

		for _, f := range files {
			os.Remove(f)
			deletions++
		}
	}

	return deletions
}

// FileReadLines returns slice of file lines for given path file
func FileReadLines(path string) (lines []string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return
}

// FileWriteLines writes given slice of lines into given path file
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
