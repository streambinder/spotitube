package cui

import (
	"container/list"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
	"github.com/lunixbochs/vtclean"
	"github.com/streambinder/spotitube/logger"
)

// CUI represents the graphical UI handler
type CUI struct {
	*gocui.Gui
	Options           Options
	Ops               *list.List
	OpsMutex          *sync.Mutex
	ProgressOffset    float64
	ProgressMax       int
	ProgressSprint    string
	PromptInputChan   chan string
	PromptDismissChan chan bool
	PromptMutex       *sync.Mutex
	Logger            *logger.Logger
	CloseChan         chan bool
}

// Operation is an enqueued UI operation
type Operation struct {
	Message string
	Options Options
}

// Startup generates a new UI handler
func Startup(options Options) (*CUI, error) {
	c := &CUI{
		Gui:            nil,
		Options:        options,
		Ops:            list.New(),
		OpsMutex:       &sync.Mutex{},
		ProgressOffset: 0,
		ProgressMax:    0,
		ProgressSprint: color.New(color.BgWhite).SprintFunc()(" "),
		PromptMutex:    &sync.Mutex{},
		Logger:         nil,
		CloseChan:      make(chan bool),
	}
	defer func() {
		go c.opsWorker()
	}()

	if !hasOption(options, GuiBareMode) {
		gui, err := gocui.NewGui(gocui.OutputNormal)
		if err != nil {
			return c, err
		}

		c.Gui = gui
		c.SetManagerFunc(func(gui *gocui.Gui) error {
			width, height := gui.Size()
			if view, err := gui.SetView(strconv.FormatUint(PanelLeftTop, 10), 0, 0,
				width/3, height/2); err != nil {
				if err != gocui.ErrUnknownView {
					return err
				}
				view.Autoscroll = true
				view.Title = strings.ToUpper(" SpotiTube ")
				fmt.Fprint(view, "\n")
			}
			if view, err := gui.SetView(strconv.FormatUint(PanelLeftBottom, 10), 0, height/2+1,
				width/3, height-4); err != nil {
				if err != gocui.ErrUnknownView {
					return err
				}
				view.Autoscroll = true
				view.Title = strings.ToUpper(" Informations ")
				fmt.Fprint(view, "\n")
			}
			if view, err := gui.SetView(strconv.FormatUint(PanelRight, 10), width/3+1, 0,
				width-1, height-4); err != nil {
				if err != gocui.ErrUnknownView {
					return err
				}
				view.Autoscroll = true
				view.Title = strings.ToUpper(" Status ")
				fmt.Fprint(view, "\n")
			}
			if _, err := gui.SetView(strconv.FormatUint(_ProgressBar, 10), 0, height-3,
				width-1, height-1); err != nil {
				if err != gocui.ErrUnknownView {
					return err
				}
			}
			return nil
		})

		if err := c.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, c.Shutdown); err != nil {
			return c, err
		}

		go func(c *CUI) {
			defer c.Close()
			if err := c.MainLoop(); err != nil {
				if err != gocui.ErrQuit {
					log.Panicln(err)
				}
			}
		}(c)
	}

	if hasOption(options, LogEnable) {
		c.Logger = logger.Build()
	}

	return c, nil
}

// Shutdown gracefully closes the graphical UI
func (c *CUI) Shutdown(gui *gocui.Gui, view *gocui.View) error {
	if c.hasOption(LogEnable) {
		c.Logger.Destroy()
	}

	c.CloseChan <- true
	if c.Gui == nil {
		return nil
	}

	c.Gui.DeleteKeybinding("", gocui.KeyCtrlC, gocui.ModNone)
	return gocui.ErrQuit
}

// OnShutdown subscribes function f to the shutdown event
func (c *CUI) OnShutdown(f func()) {
	go func() {
		<-c.CloseChan
		f()
	}()
}

// Append adds given message to given Options driven space
func (c *CUI) Append(message string, spreadOptions ...Options) error {
	var options uint64
	if len(spreadOptions) > 0 {
		options = spreadOptions[0]
	}

	c.OpsMutex.Lock()
	c.Ops.PushBack(Operation{message, options})
	defer c.OpsMutex.Unlock()

	return nil
}

func (c *CUI) view(name uint64) (*gocui.View, error) {
	return c.Gui.View(viewName(name))
}

func (c *CUI) opsWorker() {
	for true {
		c.OpsMutex.Lock()
		operation := c.Ops.Front()
		c.OpsMutex.Unlock()
		if operation != nil {
			c.opWorker(operation.Value.(Operation))
			c.OpsMutex.Lock()
			c.Ops.Remove(operation)
			c.OpsMutex.Unlock()
		}
		time.Sleep(1 * time.Microsecond)
	}
}

func (c *CUI) opWorker(operation Operation) error {
	if !c.hasOption(GuiDebugMode) && operation.hasOption(DebugAppend) {
		return nil
	}

	if c.hasOption(LogEnable) {
		c.Logger.Append(operation.Message)
	}

	if operation.hasOption(ErrorAppend) {
		operation.Options |= ColorRed | ParagraphAutoReturn
	} else if operation.hasOption(WarningAppend) {
		operation.Options |= ColorYellow | ParagraphAutoReturn
	} else if operation.hasOption(DebugAppend) {
		operation.Options |= ColorMagenta | ParagraphAutoReturn
	}

	if c.hasOption(GuiBareMode) {
		fmt.Println(vtclean.Clean(operation.Message, false))
		return nil
	}

	view, err := c.view(selectPanel(operation.Options))
	if err != nil {
		return err
	}
	if operation.hasOption(ClearAppend) {
		view.Clear()
	}
	c.Update(func(gui *gocui.Gui) error {
		var message = operation.Message
		message = Font(message, operation.Options)
		message = Color(message, operation.Options)
		message = styleParagraph(message, operation.Options, view)
		message = styleOrientation(message, operation.Options, view)
		fmt.Fprintln(view, " "+message)
		return nil
	})
	return nil
}

func selectPanel(options Options) Option {
	var panel Option = PanelRight
	if hasOption(options, PanelLeftTop) {
		panel = PanelLeftTop
	} else if hasOption(options, PanelLeftBottom) {
		panel = PanelLeftBottom
	}
	return panel
}

func viewName(name uint64) string {
	return strconv.FormatUint(name, 10)
}
