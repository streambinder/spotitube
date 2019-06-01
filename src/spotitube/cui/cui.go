package cui

import (
	"container/list"
	"fmt"
	"log"
	"math"
	"spotitube/logger"
	"spotitube/system"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

// Options : alias to uint64
type Options = uint64

// Option : alias to GuiOptions, used only for readability purposes
type Option = Options

// CUI : struct object containing all the informations to handle CUI
type CUI struct {
	*gocui.Gui
	Options           Options
	Width             int
	Height            int
	Ops               *list.List
	OpsMutex          *sync.Mutex
	LoadingOffset     float64
	LoadingMax        int
	LoadingSprint     string
	PromptInputChan   chan string
	PromptDismissChan chan bool
	PromptMutex       *sync.Mutex
	Logger            *logger.Logger
	CloseChan         chan bool
}

// Operation : enqueued GUI operation (append)
type Operation struct {
	Message string
	Options Options
}

const (
	// OptionNil : identifier for no option
	OptionNil = 1 << iota
	_
	// PromptDismissable : identifier dismissable prompt
	PromptDismissable
	// PromptDismissableWithExit : identifier dismissable with exiting prompt
	PromptDismissableWithExit
	_
	// ClearAppend : identifier for pre-append panel clearing
	ClearAppend
	// ErrorAppend : identifier for error mode append
	ErrorAppend
	// WarningAppend : identifier for warning mode append
	WarningAppend
	// DebugAppend : identifier for debug mode append
	DebugAppend
	_
	// PanelLeftTop : identifier for panel at left-top
	PanelLeftTop
	// PanelLeftBottom : identifier for panel at left-bottom
	PanelLeftBottom
	// PanelRight : identifier for panel at right
	PanelRight
	// PanelLoading : identifier for loading panel
	PanelLoading
	_
	// OrientationLeft : identifier for text left orientation
	OrientationLeft
	// OrientationCenter : identifier for text center orientation
	OrientationCenter
	// OrientationRight : identifier for text right orientation
	OrientationRight
	_
	// FontColorBlack : identifier for text black color
	FontColorBlack
	// FontColorRed : identifier for text red color
	FontColorRed
	// FontColorGreen : identifier for text green color
	FontColorGreen
	// FontColorYellow : identifier for text yellow color
	FontColorYellow
	// FontColorBlue : identifier for text blue color
	FontColorBlue
	// FontColorMagenta : identifier for text magenta color
	FontColorMagenta
	// FontColorCyan : identifier for text cyan color
	FontColorCyan
	// FontColorWhite : identifier for text white color
	FontColorWhite
	_
	// FontStyleBold : identifier for text bold style
	FontStyleBold
	// FontStyleUnderline : identifier for text underline style
	FontStyleUnderline
	// FontStyleReverse : identifier for text reverse style
	FontStyleReverse
	_
	// ParagraphStyleStandard : identifier for text standard paragraph format
	ParagraphStyleStandard
	// ParagraphStyleAutoReturn : identifier for text autoreturning paragraph format, to fit words in lines
	ParagraphStyleAutoReturn
	_
	// LogEnable : identifier for log writing flag
	LogEnable
	_
	// GuiBareMode : identifier for make gui as bare as possible
	GuiBareMode
	// GuiDebugMode : identifier for enabling debug mode
	GuiDebugMode
)

var (
	// Panels : all panels identifiers to real names mapping
	Panels = map[uint64]string{
		PanelLeftTop:    "GuiPanelLeftTop",
		PanelLeftBottom: "GuiPanelLeftBottom",
		PanelRight:      "GuiPanelRight",
		PanelLoading:    "GuiPanelLoading",
	}
	// FontColors : all text colors identifiers to auxiliary library values mapping
	FontColors = map[uint64]color.Attribute{
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
	FontStyles = map[uint64]color.Attribute{
		FontStyleBold: color.Bold,
	}
)

// Startup : generate a Gui object
func Startup(options Options) (*CUI, error) {
	c := &CUI{
		Options:       options,
		Ops:           list.New(),
		OpsMutex:      &sync.Mutex{},
		LoadingOffset: 0,
		LoadingMax:    0,
		LoadingSprint: color.New(color.BgWhite).SprintFunc()(" "),
		// PromptInputChan:   make(chan string),
		// PromptDismissChan: make(chan bool),
		PromptMutex: &sync.Mutex{},
		Logger:      nil,
		CloseChan:   make(chan bool),
	}
	defer func() {
		go guiDequeueOps(c)
	}()

	if !hasOption(options, GuiBareMode) {
		gui, err := gocui.NewGui(gocui.OutputNormal)
		if err != nil {
			return c, err
		}

		c.Gui = gui
		c.SetManagerFunc(guiStandardLayout)
		c.Width, c.Height = c.Size()

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
		c.Logger = logger.Build(logger.DefaultLogFname)
	}

	return c, nil
}

// Shutdown : shut down the interface
func (c *CUI) Shutdown(gui *gocui.Gui, view *gocui.View) error {
	c.CloseChan <- true
	c.Gui.DeleteKeybinding("", gocui.KeyCtrlC, gocui.ModNone)
	return gocui.ErrQuit
}

// Append : add input string message to input Options driven space
func (c *CUI) Append(message string, options ...Options) error {
	var firstOptions uint64
	if len(options) > 0 {
		firstOptions = options[0]
	}

	c.OpsMutex.Lock()
	c.Ops.PushBack(Operation{message, firstOptions})
	defer c.OpsMutex.Unlock()

	return nil
}

// Prompt : show a prompt containing input string message, driven with input Options
func (c *CUI) Prompt(message string, options Options) error {
	if c.hasOption(GuiBareMode) {
		fmt.Println(message)
		if hasOption(options, PromptDismissableWithExit) {
			c.CloseChan <- true
		}
		return nil
	}

	c.PromptMutex.Lock()
	defer c.PromptMutex.Unlock()

	if c.hasOption(LogEnable) {
		c.Logger.Append(message)
	}

	c.PromptDismissChan = make(chan bool)
	c.Update(func(gui *gocui.Gui) error {
		var (
			view *gocui.View
			err  error
		)
		guiWidth, guiHeight := gui.Size()
		if view, err = gui.SetView("GuiPrompt",
			guiWidth/2-(len(message)/2)-2, guiHeight/2,
			guiWidth/2+(len(message)/2), guiHeight/2+2); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			fmt.Fprintln(view, message)
			if hasOption(options, PromptDismissableWithExit) {
				gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, c.callbackShutdown)
			} else {
				gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, c.callback)
			}
		}
		return nil
	})
	<-c.PromptDismissChan
	return nil
}

// PromptInput : show a confirmation/cancel prompt containing input string message, driven with input Options
func (c *CUI) PromptInput(message string, options Options) bool {
	if c.hasOption(GuiBareMode) {
		return system.InputConfirm(message)
	}

	c.PromptMutex.Lock()
	defer c.PromptMutex.Unlock()

	c.PromptDismissChan = make(chan bool)
	c.Update(func(gui *gocui.Gui) error {
		var (
			view *gocui.View
			err  error
		)
		message = fmt.Sprintf("%s\n\nPress TAB to cancel, ENTER to confirm.", message)
		guiWidth, guiHeight := gui.Size()
		neededWidth, neededHeight := 0, strings.Count(message, "\n")
		for _, line := range strings.Split(message, "\n") {
			if len(line) > neededWidth {
				neededWidth = len(line)
			}
		}
		neededWidth += 2
		if view, err = gui.SetView("GuiPrompt",
			guiWidth/2-(int(neededWidth/2))-2, (guiHeight/2)-int(math.Floor(float64(neededHeight/2))),
			guiWidth/2+(int(neededWidth/2)), (guiHeight/2)+int(math.Ceil(float64(neededHeight/2)))+3); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, c.callbackInputConfirm)
			gui.SetKeybinding("", gocui.KeyTab, gocui.ModNone, c.callbackInputCancel)
			fmt.Fprintln(view, messageOrientate(message, view, OrientationCenter))
		}
		return nil
	})
	return <-c.PromptDismissChan
}

// PromptInputMessage : show an input prompt containing input string message, driven with input Options
func (c *CUI) PromptInputMessage(message string, options Options) string {
	if c.hasOption(GuiBareMode) {
		return system.InputString(message)
	}

	c.PromptMutex.Lock()
	defer c.PromptMutex.Unlock()

	c.PromptInputChan = make(chan string)
	c.Update(func(gui *gocui.Gui) error {
		var (
			view *gocui.View
			err  error
		)
		guiWidth, guiHeight := gui.Size()
		if view, err = gui.SetView("GuiPrompt",
			guiWidth/2-50, (guiHeight/2)-1,
			guiWidth/2+50, (guiHeight/2)+1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			view.Editable = true
			view.Title = fmt.Sprintf(" %s ", message)
			_ = view.SetCursor(0, 0)
			gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, c.callbackInputTyping)
			_, _ = gui.SetCurrentView("GuiPrompt")
		}
		return nil
	})
	return strings.Replace(<-c.PromptInputChan, "\n", "", -1)
}

// LoadingFill : fill up the bottom loading bar
func (c *CUI) LoadingFill() error {
	if c.hasOption(GuiBareMode) {
		return nil
	}

	c.Update(func(gui *gocui.Gui) error {
		view, err := gui.View(Panels[PanelLoading])
		if err != nil {
			return err
		}
		maxWidth, _ := view.Size()
		view.Clear()
		view.Title = fmt.Sprintf(" 100 %% ")
		fmt.Fprint(view, strings.Repeat(c.LoadingSprint, maxWidth))
		return nil
	})
	return nil
}

// LoadingIncrease : increase loading bar
func (c *CUI) LoadingIncrease() error {
	if c.hasOption(GuiBareMode) {
		return nil
	}

	c.Update(func(gui *gocui.Gui) error {
		view, err := gui.View(Panels[PanelLoading])
		if err != nil {
			return err
		}
		maxWidth, _ := view.Size()
		view.Clear()
		view.Title = fmt.Sprintf(" %d %% ", int(math.Floor(c.LoadingOffset))*100/c.LoadingMax)
		fmt.Fprint(view, strings.Repeat(c.LoadingSprint, int(math.Floor(c.LoadingOffset))*maxWidth/c.LoadingMax))
		c.LoadingOffset++
		return nil
	})
	return nil
}

// LoadingHalfIncrease : increase loading bar by half-step
func (c *CUI) LoadingHalfIncrease() error {
	if c.hasOption(GuiBareMode) {
		return nil
	}

	c.Update(func(gui *gocui.Gui) error {
		view, err := gui.View(Panels[PanelLoading])
		if err != nil {
			return err
		}
		maxWidth, _ := view.Size()
		view.Clear()
		view.Title = fmt.Sprintf(" %d %% ", int(math.Floor(c.LoadingOffset))*100/c.LoadingMax)
		fmt.Fprint(view, strings.Repeat(c.LoadingSprint, int(math.Floor(c.LoadingOffset))*maxWidth/c.LoadingMax))
		c.LoadingOffset += 0.5
		return nil
	})
	return nil
}

// MessageStyle : apply input uint64 styleConst styling to input string message
func MessageStyle(message string, styleConst uint64) string {
	styleFunc := color.New(FontStyles[styleConst])
	return styleFunc.Sprintf(message)
}

func (c *CUI) opHandle(operation Operation) error {
	if !c.hasOption(GuiDebugMode) && operation.hasOption(DebugAppend) {
		return nil
	}

	if c.hasOption(LogEnable) {
		c.Logger.Append(operation.Message)
	}

	if operation.hasOption(ErrorAppend) {
		operation.Options = operation.Options | FontColorRed | ParagraphStyleAutoReturn
	} else if operation.hasOption(WarningAppend) {
		operation.Options = operation.Options | FontColorYellow | ParagraphStyleAutoReturn
	} else if operation.hasOption(DebugAppend) {
		operation.Options = operation.Options | FontColorMagenta | ParagraphStyleAutoReturn
	}

	if c.hasOption(GuiBareMode) {
		fmt.Println(condColorStyle(condFontStyle(operation.Message, operation.Options), operation.Options))
		return nil
	}

	var (
		view  *gocui.View
		err   error
		panel Option
	)
	panel = condPanelSelector(operation.Options)
	view, err = c.View(Panels[panel])
	if err != nil {
		return err
	}
	if operation.hasOption(ClearAppend) {
		view.Clear()
	}
	c.Update(func(gui *gocui.Gui) error {
		width, _ := view.Size()
		var message = condParagraphStyle(operation.Message, operation.Options, width)
		message = condFontStyle(operation.Message, operation.Options)
		message = condOrientationStyle(operation.Message, operation.Options, view)
		message = condColorStyle(operation.Message, operation.Options)
		fmt.Fprintln(view, " "+message)
		return nil
	})
	return nil
}

func (operation Operation) hasOption(option Option) bool {
	return hasOption(operation.Options, option)
}

func (c *CUI) hasOption(option Option) bool {
	return hasOption(c.Options, option)
}

func hasOption(options Options, option Option) bool {
	return (options & option) != 0
}

func condPanelSelector(options Options) Option {
	var panel Option = PanelRight
	if hasOption(options, PanelLeftTop) {
		panel = PanelLeftTop
	} else if hasOption(options, PanelLeftBottom) {
		panel = PanelLeftBottom
	}
	return panel
}

func condParagraphStyle(message string, options Option, width int) string {
	if hasOption(options, ParagraphStyleStandard) {
		message = messageParagraphStyle(message, ParagraphStyleStandard, width)
	} else if hasOption(options, ParagraphStyleAutoReturn) {
		message = messageParagraphStyle(message, ParagraphStyleAutoReturn, width)
	}
	return message
}

func condFontStyle(message string, options Option) string {
	if hasOption(options, FontStyleBold) {
		message = MessageStyle(message, FontStyleBold)
	} else if hasOption(options, FontStyleUnderline) {
		message = MessageStyle(message, FontStyleUnderline)
	} else if hasOption(options, FontStyleReverse) {
		message = MessageStyle(message, FontStyleReverse)
	}
	return message
}

func condOrientationStyle(message string, options Option, view *gocui.View) string {
	if hasOption(options, OrientationLeft) {
		message = messageOrientate(message, view, OrientationLeft)
	} else if hasOption(options, OrientationCenter) {
		message = messageOrientate(message, view, OrientationCenter)
	} else if hasOption(options, OrientationRight) {
		message = messageOrientate(message, view, OrientationRight)
	}
	return message
}

func condColorStyle(message string, options Option) string {
	if hasOption(options, FontColorBlack) {
		message = messageColor(message, FontColorBlack)
	} else if hasOption(options, FontColorRed) {
		message = messageColor(message, FontColorRed)
	} else if hasOption(options, FontColorGreen) {
		message = messageColor(message, FontColorGreen)
	} else if hasOption(options, FontColorYellow) {
		message = messageColor(message, FontColorYellow)
	} else if hasOption(options, FontColorBlue) {
		message = messageColor(message, FontColorBlue)
	} else if hasOption(options, FontColorMagenta) {
		message = messageColor(message, FontColorMagenta)
	} else if hasOption(options, FontColorCyan) {
		message = messageColor(message, FontColorCyan)
	} else if hasOption(options, FontColorWhite) {
		message = messageColor(message, FontColorWhite)
	}
	return message
}

func messageOrientate(message string, view *gocui.View, orientation int) string {
	var messageLines []string
	var lineLength, _ = view.Size()
	for _, line := range strings.Split(message, "\n") {
		if len(line) < lineLength {
			lineSpacing := (lineLength - len(line)) / 2
			if orientation == OrientationCenter {
				line = strings.Repeat(" ", lineSpacing) +
					line + strings.Repeat(" ", lineSpacing)
			} else if orientation == OrientationRight {
				line = strings.Repeat(" ", lineSpacing*2-1) + line
			}
		}
		messageLines = append(messageLines, line)
	}
	return strings.Join(messageLines, "\n")
}

func messageColor(message string, colorConst uint64) string {
	colorFunc := color.New(FontColors[colorConst])
	return colorFunc.Sprintf(message)
}

func messageParagraphStyle(message string, styleConst uint64, width int) string {
	if styleConst == ParagraphStyleAutoReturn {
		var messageParagraph string
		for len(message) > 0 {
			if len(message) < width {
				messageParagraph = messageParagraph + message
				message = ""
			} else {
				messageParagraph = messageParagraph + message[:width] + "\n"
				message = message[width:]
			}
		}
		return messageParagraph
	}
	return message
}

func (c *CUI) callback(gui *gocui.Gui, view *gocui.View) error {
	c.Update(func(gui *gocui.Gui) error {
		gui.DeleteView("GuiPrompt")
		return nil
	})
	c.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	c.PromptDismissChan <- true
	return nil
}

func (c *CUI) callbackShutdown(gui *gocui.Gui, view *gocui.View) error {
	c.callback(c.Gui, view)
	return guiClose(c.Gui, view)
}

func (c *CUI) callbackInputInteractive(gui *gocui.Gui, view *gocui.View) error {
	c.Update(func(gui *gocui.Gui) error {
		gui.DeleteView("GuiPrompt")
		return nil
	})
	c.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	c.DeleteKeybinding("", gocui.KeyTab, gocui.ModNone)
	time.Sleep(1 * time.Millisecond)
	return nil
}

func (c *CUI) callbackInputConfirm(gui *gocui.Gui, view *gocui.View) error {
	if err := c.callbackInputInteractive(c.Gui, view); err != nil {
		return err
	}
	c.PromptDismissChan <- true
	return nil
}

func (c *CUI) callbackInputCancel(gui *gocui.Gui, view *gocui.View) error {
	if err := c.callbackInputInteractive(c.Gui, view); err != nil {
		return err
	}
	c.PromptDismissChan <- false
	return nil
}

func (c *CUI) callbackInputTyping(gui *gocui.Gui, view *gocui.View) error {
	c.Update(func(gui *gocui.Gui) error {
		gui.DeleteView("GuiPrompt")
		return nil
	})
	c.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	view.Rewind()
	c.PromptInputChan <- view.Buffer()
	return nil
}

func guiDequeueOps(c *CUI) {
	for true {
		c.OpsMutex.Lock()
		operation := c.Ops.Front()
		c.OpsMutex.Unlock()
		if operation != nil {
			c.opHandle(operation.Value.(Operation))
			c.OpsMutex.Lock()
			c.Ops.Remove(operation)
			c.OpsMutex.Unlock()
		}
		time.Sleep(1 * time.Microsecond)
	}
}

func guiStandardLayout(gui *gocui.Gui) error {
	guiMaxWidth, guiMaxHeight := gui.Size()
	if view, err := gui.SetView("GuiPanelLeftTop", 0, 0,
		guiMaxWidth/3, guiMaxHeight/2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		view.Autoscroll = true
		view.Title = strings.ToUpper(" SpotiTube ")
		fmt.Fprint(view, "\n")
	}
	if view, err := gui.SetView("GuiPanelLeftBottom", 0, guiMaxHeight/2+1,
		guiMaxWidth/3, guiMaxHeight-4); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		view.Autoscroll = true
		view.Title = strings.ToUpper(" Informations ")
		fmt.Fprint(view, "\n")
	}
	if view, err := gui.SetView("GuiPanelRight", guiMaxWidth/3+1, 0,
		guiMaxWidth-1, guiMaxHeight-4); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		view.Autoscroll = true
		view.Title = strings.ToUpper(" Status ")
		fmt.Fprint(view, "\n")
	}
	if _, err := gui.SetView("GuiPanelLoading", 0, guiMaxHeight-3,
		guiMaxWidth-1, guiMaxHeight-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}
	return nil
}

func guiClose(gui *gocui.Gui, view *gocui.View) error {
	// singleton.Closing <- true
	gui.DeleteKeybinding("", gocui.KeyCtrlC, gocui.ModNone)
	return gocui.ErrQuit
}
