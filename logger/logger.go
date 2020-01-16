package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lunixbochs/vtclean"
)

var (
	filePath = fmt.Sprintf("spotitube_%s.log", time.Now().Format("2006-01-02_15.04.05"))
)

// Logger contains all the information kept to handle logging
type Logger struct {
	filePath   string
	fileHandle *os.File
	mutex      sync.Mutex
}

// Build returns a new logger
func Build() *Logger {
	return &Logger{
		filePath: filePath,
	}
}

// Append writes a new log line with given message
func (log *Logger) Append(message string) {
	go func() error {
		log.mutex.Lock()
		defer log.mutex.Unlock()

		if log.fileHandle == nil {
			fileHandle, err := os.OpenFile(log.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			log.fileHandle = fileHandle
		}

		if _, err := log.fileHandle.WriteString(
			fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"),
				vtclean.Clean(strings.Replace(message, "\n", " ", -1), false))); err != nil {
			return err
		}

		return nil
	}()
}

// Destroy close the file descriptor corresponding to the log
func (log *Logger) Destroy() error {
	if log.fileHandle != nil {
		return log.fileHandle.Close()
	}
	return nil
}
