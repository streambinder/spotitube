package logger

import (
	"fmt"
	"time"
)

var (
	// DefaultLogFname : default log filename
	DefaultLogFname = fmt.Sprintf("spotitube_%s.log", time.Now().Format("2006-01-02_15.04.05"))
)
