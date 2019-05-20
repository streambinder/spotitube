package gui

import (
	spttb_logger "logger"

	"github.com/jroimartin/gocui"
)

// Options : alias to uint64
type Options = uint64

// Option : alias to GuiOptions, used only for readability purposes
type Option = Options

// Gui : struct object containing all the informations to handle GUI
type Gui struct {
	*gocui.Gui
	Width   int
	Height  int
	Options Options
	Closing chan bool
	Logger  *spttb_logger.Logger
}

// Operation : enqueued GUI operation (append)
type Operation struct {
	Message string
	Options Options
}
