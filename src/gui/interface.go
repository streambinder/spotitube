package gui

import (
	"fmt"
	"log"
	"math"
	"strings"

	spttb_logger "logger"
	spttb_system "system"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

// Build : generate a Gui object
func Build(options uint64) *Gui {
	if (options & GuiSilentMode) == 0 {
		var gui *gocui.Gui
		guiReady = make(chan *gocui.Gui)
		go subGuiRun()
		gui = <-guiReady
		guiWidth, guiHeight := gui.Size()

		singleton = &Gui{
			gui,
			guiWidth,
			guiHeight,
			options,
			make(chan bool),
			nil,
		}
		return singleton
	}

	return &Gui{
		&gocui.Gui{},
		0,
		0,
		options,
		make(chan bool),
		nil,
	}
}

// LinkLogger : link input Logger logger to Gui
func (gui *Gui) LinkLogger(logger *spttb_logger.Logger) error {
	gui.Logger = logger
	return nil
}

// Append : add input string message to input uint64 options driven space
func (gui *Gui) Append(message string, options uint64) error {
	if (gui.Options & GuiSilentMode) != 0 {
		fmt.Println(message)
		return nil
	}

	guiAppendMutex.Lock()
	defer guiAppendMutex.Unlock()

	if (options&LogNoWrite) == 0 && gui.Logger != nil {
		gui.Logger.Append(message)
	}

	var (
		view  *gocui.View
		err   error
		panel uint64
	)
	panel = subCondPanelSelector(options)
	view, err = gui.View(Panels[int(panel)])
	if err != nil {
		return err
	}
	gui.Update(func(gui *gocui.Gui) error {
		width, _ := view.Size()
		message = subCondParagraphStyle(message, options, width)
		message = subCondFontStyle(message, options)
		message = subCondOrientationStyle(message, options, view)
		message = subCondColorStyle(message, options)
		fmt.Fprintln(view, " "+message)
		return nil
	})
	return nil
}

// ClearAppend : add input string message to input uint64 options driven space, after clearing its container
func (gui *Gui) ClearAppend(message string, options uint64) error {
	if (gui.Options & GuiSilentMode) != 0 {
		fmt.Println(message)
		return nil
	}

	var (
		view  *gocui.View
		err   error
		panel uint64
	)
	if (options & PanelRight) != 0 {
		panel = PanelRight
	} else if (options & PanelLeftTop) != 0 {
		panel = PanelLeftTop
	} else if (options & PanelLeftBottom) != 0 {
		panel = PanelLeftBottom
	}
	view, err = gui.View(Panels[int(panel)])
	if err != nil {
		return err
	}
	view.Clear()
	return gui.Append(message, options|panel)
}

// ErrAppend : add input string message, formatted as error, to input uint64 options driven space
func (gui *Gui) ErrAppend(message string, options uint64) error {
	return gui.Append(message, options|FontColorRed|ParagraphStyleAutoReturn)
}

// WarnAppend : add input string message, formatted as warning, to input uint64 options driven space
func (gui *Gui) WarnAppend(message string, options uint64) error {
	return gui.Append(message, options|FontColorYellow|ParagraphStyleAutoReturn)
}

// DebugAppend : add input string message, formatted as debug message, to input uint64 options driven space
func (gui *Gui) DebugAppend(message string, options uint64) error {
	if (gui.Options & GuiDebugMode) == 0 {
		return nil
	}
	return gui.Append(message, options|ParagraphStyleAutoReturn|FontColorMagenta)
}

// Prompt : show a prompt containing input string message, driven with input uint64 options
func (gui *Gui) Prompt(message string, options uint64) error {
	if (gui.Options & GuiSilentMode) != 0 {
		fmt.Println(message)
		return nil
	}

	guiPromptMutex.Lock()
	defer guiPromptMutex.Unlock()

	guiPromptDismiss = make(chan bool)
	if (options&LogNoWrite) == 0 && gui.Logger != nil {
		gui.Logger.Append(message)
	}
	gui.Update(func(gui *gocui.Gui) error {
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
			if (options & PromptDismissableWithExit) != 0 {
				gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, subGuiDismissPromptAndClose)
			} else {
				gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, subGuiDismissPrompt)
			}
		}
		return nil
	})
	<-guiPromptDismiss
	return nil
}

// PromptInput : show a confirmation/cancel prompt containing input string message, driven with input uint64 options
func (gui *Gui) PromptInput(message string, options uint64) bool {
	if (gui.Options & GuiSilentMode) != 0 {
		return spttb_system.InputConfirm(message)
	}

	guiPromptMutex.Lock()
	defer guiPromptMutex.Unlock()

	guiPromptDismiss = make(chan bool)
	gui.Update(func(gui *gocui.Gui) error {
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
			gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, subGuiDismissPromptWithInputOk)
			gui.SetKeybinding("", gocui.KeyTab, gocui.ModNone, subGuiDismissPromptWithInputNok)
			fmt.Fprintln(view, subMessageOrientate(message, view, OrientationCenter))
		}
		return nil
	})
	return <-guiPromptDismiss
}

// PromptInputMessage : show an input prompt containing input string message, driven with input uint64 options
func (gui *Gui) PromptInputMessage(message string, options uint64) string {
	if (gui.Options & GuiSilentMode) != 0 {
		return spttb_system.InputString(message)
	}

	guiPromptMutex.Lock()
	defer guiPromptMutex.Unlock()

	guiPromptInput = make(chan string)
	gui.Update(func(gui *gocui.Gui) error {
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
			gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, subGuiDismissPromptWithInputMessage)
			_, _ = gui.SetCurrentView("GuiPrompt")
		}
		return nil
	})
	return strings.Replace(<-guiPromptInput, "\n", "", -1)
}

// LoadingSetMax : set maximum value for bottom loading bar
func (gui *Gui) LoadingSetMax(max int) error {
	guiLoadingMax = max
	return nil
}

// LoadingFill : fill up the bottom loading bar
func (gui *Gui) LoadingFill() error {
	if (gui.Options & GuiSilentMode) != 0 {
		return nil
	}

	gui.Update(func(gui *gocui.Gui) error {
		view, err := gui.View(Panels[PanelLoading])
		if err != nil {
			return err
		}
		maxWidth, _ := view.Size()
		view.Clear()
		view.Title = fmt.Sprintf(" 100 %% ")
		fmt.Fprint(view, strings.Repeat(guiLoadingSprint, maxWidth))
		return nil
	})
	return nil
}

// LoadingIncrease : increase loading bar
func (gui *Gui) LoadingIncrease() error {
	if (gui.Options & GuiSilentMode) != 0 {
		return nil
	}

	gui.Update(func(gui *gocui.Gui) error {
		view, err := gui.View(Panels[PanelLoading])
		if err != nil {
			return err
		}
		maxWidth, _ := view.Size()
		view.Clear()
		view.Title = fmt.Sprintf(" %d %% ", int(math.Floor(guiLoadingCtr))*100/guiLoadingMax)
		fmt.Fprint(view, strings.Repeat(guiLoadingSprint, int(math.Floor(guiLoadingCtr))*maxWidth/guiLoadingMax))
		guiLoadingCtr++
		return nil
	})
	return nil
}

// LoadingHalfIncrease : increase loading bar by half-step
func (gui *Gui) LoadingHalfIncrease() error {
	if (gui.Options & GuiSilentMode) != 0 {
		return nil
	}

	gui.Update(func(gui *gocui.Gui) error {
		view, err := gui.View(Panels[PanelLoading])
		if err != nil {
			return err
		}
		maxWidth, _ := view.Size()
		view.Clear()
		view.Title = fmt.Sprintf(" %d %% ", int(math.Floor(guiLoadingCtr))*100/guiLoadingMax)
		fmt.Fprint(view, strings.Repeat(guiLoadingSprint, int(math.Floor(guiLoadingCtr))*maxWidth/guiLoadingMax))
		guiLoadingCtr += 0.5
		return nil
	})
	return nil
}

// MessageStyle : apply input int styleConst styling to input string message
func MessageStyle(message string, styleConst int) string {
	styleFunc := color.New(FontStyles[styleConst])
	return styleFunc.Sprintf(message)
}

func subCondPanelSelector(options uint64) uint64 {
	var panel uint64
	if (options & PanelLeftTop) != 0 {
		panel = PanelLeftTop
	} else if (options & PanelLeftBottom) != 0 {
		panel = PanelLeftBottom
	} else {
		panel = PanelRight
	}
	return panel
}

func subCondParagraphStyle(message string, options uint64, width int) string {
	if (options & ParagraphStyleStandard) != 0 {
		message = subMessageParagraphStyle(message, ParagraphStyleStandard, width)
	} else if (options & ParagraphStyleAutoReturn) != 0 {
		message = subMessageParagraphStyle(message, ParagraphStyleAutoReturn, width)
	}
	return message
}

func subCondFontStyle(message string, options uint64) string {
	if (options & FontStyleBold) != 0 {
		message = MessageStyle(message, FontStyleBold)
	} else if (options & FontStyleUnderline) != 0 {
		message = MessageStyle(message, FontStyleUnderline)
	} else if (options & FontStyleReverse) != 0 {
		message = MessageStyle(message, FontStyleReverse)
	}
	return message
}

func subCondOrientationStyle(message string, options uint64, view *gocui.View) string {
	if (options & OrientationLeft) != 0 {
		message = subMessageOrientate(message, view, OrientationLeft)
	} else if (options & OrientationCenter) != 0 {
		message = subMessageOrientate(message, view, OrientationCenter)
	} else if (options & OrientationRight) != 0 {
		message = subMessageOrientate(message, view, OrientationRight)
	}
	return message
}

func subCondColorStyle(message string, options uint64) string {
	if (options & FontColorBlack) != 0 {
		message = subMessageColor(message, FontColorBlack)
	} else if (options & FontColorRed) != 0 {
		message = subMessageColor(message, FontColorRed)
	} else if (options & FontColorGreen) != 0 {
		message = subMessageColor(message, FontColorGreen)
	} else if (options & FontColorYellow) != 0 {
		message = subMessageColor(message, FontColorYellow)
	} else if (options & FontColorBlue) != 0 {
		message = subMessageColor(message, FontColorBlue)
	} else if (options & FontColorMagenta) != 0 {
		message = subMessageColor(message, FontColorMagenta)
	} else if (options & FontColorCyan) != 0 {
		message = subMessageColor(message, FontColorCyan)
	} else if (options & FontColorWhite) != 0 {
		message = subMessageColor(message, FontColorWhite)
	}
	return message
}

func subMessageOrientate(message string, view *gocui.View, orientation int) string {
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

func subMessageColor(message string, colorConst int) string {
	colorFunc := color.New(FontColors[colorConst])
	return colorFunc.Sprintf(message)
}

func subMessageParagraphStyle(message string, styleConst int, width int) string {
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

func subGuiDismissPrompt(gui *gocui.Gui, view *gocui.View) error {
	gui.Update(func(gui *gocui.Gui) error {
		gui.DeleteView("GuiPrompt")
		return nil
	})
	gui.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	guiPromptDismiss <- true
	return nil
}

func subGuiDismissPromptAndClose(gui *gocui.Gui, view *gocui.View) error {
	subGuiDismissPrompt(gui, view)
	return subGuiClose(gui, view)
}

func subGuiDismissPromptWithInput(gui *gocui.Gui, view *gocui.View) error {
	gui.Update(func(gui *gocui.Gui) error {
		gui.DeleteView("GuiPrompt")
		return nil
	})
	gui.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	gui.DeleteKeybinding("", gocui.KeyTab, gocui.ModNone)
	return nil
}

func subGuiDismissPromptWithInputOk(gui *gocui.Gui, view *gocui.View) error {
	if err := subGuiDismissPromptWithInput(gui, view); err != nil {
		return err
	}
	guiPromptDismiss <- true
	return nil
}

func subGuiDismissPromptWithInputNok(gui *gocui.Gui, view *gocui.View) error {
	if err := subGuiDismissPromptWithInput(gui, view); err != nil {
		return err
	}
	guiPromptDismiss <- false
	return nil
}

func subGuiDismissPromptWithInputMessage(gui *gocui.Gui, view *gocui.View) error {
	gui.Update(func(gui *gocui.Gui) error {
		gui.DeleteView("GuiPrompt")
		return nil
	})
	gui.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	view.Rewind()
	guiPromptInput <- view.Buffer()
	return nil
}

func subGuiRun() {
	gui, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer gui.Close()

	gui.SetManagerFunc(subGuiStandardLayout)

	if err := gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, subGuiClose); err != nil {
		log.Panicln(err)
	}

	guiReady <- gui

	if err := gui.MainLoop(); err != nil {
		if err != gocui.ErrQuit {
			log.Panicln(err)
		}
	}
}

func subGuiStandardLayout(gui *gocui.Gui) error {
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

func subGuiClose(gui *gocui.Gui, view *gocui.View) error {
	singleton.Closing <- true
	gui.DeleteKeybinding("", gocui.KeyCtrlC, gocui.ModNone)
	return gocui.ErrQuit
}
