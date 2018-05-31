package logger

import (
	"sync"
)

// Logger : struct containing all the informations kept to handle logging
type Logger struct {
	File  string
	Mutex sync.Mutex
}
