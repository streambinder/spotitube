package gui

import (
	"fmt"
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
		go guiRun()
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
	panel = condPanelSelector(options)
	view, err = gui.View(Panels[int(panel)])
	if err != nil {
		return err
	}
	gui.Update(func(gui *gocui.Gui) error {
		width, _ := view.Size()
		message = condParagraphStyle(message, options, width)
		message = condFontStyle(message, options)
		message = condOrientationStyle(message, options, view)
		message = condColorStyle(message, options)
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
				gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, guiDismissPromptAndClose)
			} else {
				gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, guiDismissPrompt)
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
			gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, guiDismissPromptWithInputOk)
			gui.SetKeybinding("", gocui.KeyTab, gocui.ModNone, guiDismissPromptWithInputNok)
			fmt.Fprintln(view, messageOrientate(message, view, OrientationCenter))
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
			gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, guiDismissPromptWithInputMessage)
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
