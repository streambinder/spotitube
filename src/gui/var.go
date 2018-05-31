package gui

import (
	"sync"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

var (
	// Panels : all panels identifiers to real names mapping
	Panels = map[int]string{
		PanelLeftTop:    "GuiPanelLeftTop",
		PanelLeftBottom: "GuiPanelLeftBottom",
		PanelRight:      "GuiPanelRight",
		PanelLoading:    "GuiPanelLoading",
	}
	// FontColors : all text colors identifiers to auxiliary library values mapping
	FontColors = map[int]color.Attribute{
		FontColorBlack:   color.FgBlack,
		FontColorRed:     color.FgRed,
		FontColorGreen:   color.FgGreen,
		FontColorYellow:  color.FgYellow,
		FontColorBlue:    color.FgBlue,
		FontColorMagenta: color.FgMagenta,
		FontColorCyan:    color.FgCyan,
		FontColorWhite:   color.FgWhite,
	}
	// FontStyles : all text styles identifiers to auxiliary library values mapping
	FontStyles = map[int]color.Attribute{
		FontStyleBold: color.Bold,
	}

	guiReady         chan *gocui.Gui
	guiPromptDismiss chan bool
	guiPromptInput   chan string
	guiPromptMutex   sync.Mutex
	guiAppendMutex   sync.Mutex
	guiLoadingCtr    int
	guiLoadingMax    = 100
	guiLoadingSprint = color.New(color.BgWhite).SprintFunc()(" ")

	singleton *Gui
)
