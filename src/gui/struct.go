package gui

import (
	spttb_logger "logger"

	"github.com/jroimartin/gocui"
)

// Gui : struct object containing all the informations to handle GUI
type Gui struct {
	*gocui.Gui
	Width   int
	Height  int
	Verbose bool
	Closing chan bool
	Logger  *spttb_logger.Logger
}
