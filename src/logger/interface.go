package logger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lunixbochs/vtclean"
)

// Build : Logger struct object constructor
func Build(filename string) *Logger {
	return &Logger{
		File: filename,
	}
}

// Append : make Logger object log input message string, eventually throwing a returning error
func (logger *Logger) Append(message string) error {
	logger.Mutex.Lock()
	defer logger.Mutex.Unlock()
	loggerFile, err := os.OpenFile(logger.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer loggerFile.Close()
	if _, err = loggerFile.WriteString(fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"),
		vtclean.Clean(strings.Replace(message, "\n", " ", -1), false))); err != nil {
		return err
	}
	return nil
}
