package system

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

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
