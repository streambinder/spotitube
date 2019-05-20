package gui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

func (gui *Gui) opHandle(operation Operation) error {
	if !gui.hasOption(GuiDebugMode) && operation.hasOption(DebugAppend) {
		return nil
	}

	if !operation.hasOption(LogNoWrite) && gui.Logger != nil {
		gui.Logger.Append(operation.Message)
	}

	if operation.hasOption(ErrorAppend) {
		operation.Options = operation.Options | FontColorRed | ParagraphStyleAutoReturn
	} else if operation.hasOption(WarningAppend) {
		operation.Options = operation.Options | FontColorYellow | ParagraphStyleAutoReturn
	} else if operation.hasOption(DebugAppend) {
		operation.Options = operation.Options | FontColorMagenta | ParagraphStyleAutoReturn
	}

	if gui.hasOption(GuiBareMode) {
		fmt.Println(condColorStyle(condFontStyle(operation.Message, operation.Options), operation.Options))
		return nil
	}

	var (
		view  *gocui.View
		err   error
		panel Option
	)
	panel = condPanelSelector(operation.Options)
	view, err = gui.View(Panels[panel])
	if err != nil {
		return err
	}
	if operation.hasOption(ClearAppend) {
		view.Clear()
	}
	gui.Update(func(gui *gocui.Gui) error {
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

func (gui *Gui) hasOption(option Option) bool {
	return hasOption(gui.Options, option)
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

func guiDismissPrompt(gui *gocui.Gui, view *gocui.View) error {
	gui.Update(func(gui *gocui.Gui) error {
		gui.DeleteView("GuiPrompt")
		return nil
	})
	gui.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	guiPromptDismiss <- true
	return nil
}

func guiDismissPromptAndClose(gui *gocui.Gui, view *gocui.View) error {
	guiDismissPrompt(gui, view)
	return guiClose(gui, view)
}

func guiDismissPromptWithInput(gui *gocui.Gui, view *gocui.View) error {
	gui.Update(func(gui *gocui.Gui) error {
		gui.DeleteView("GuiPrompt")
		return nil
	})
	gui.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	gui.DeleteKeybinding("", gocui.KeyTab, gocui.ModNone)
	return nil
}

func guiDismissPromptWithInputOk(gui *gocui.Gui, view *gocui.View) error {
	if err := guiDismissPromptWithInput(gui, view); err != nil {
		return err
	}
	guiPromptDismiss <- true
	return nil
}

func guiDismissPromptWithInputNok(gui *gocui.Gui, view *gocui.View) error {
	if err := guiDismissPromptWithInput(gui, view); err != nil {
		return err
	}
	guiPromptDismiss <- false
	return nil
}

func guiDismissPromptWithInputMessage(gui *gocui.Gui, view *gocui.View) error {
	gui.Update(func(gui *gocui.Gui) error {
		gui.DeleteView("GuiPrompt")
		return nil
	})
	gui.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	view.Rewind()
	guiPromptInput <- view.Buffer()
	return nil
}

func guiRun() {
	gui, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer gui.Close()

	gui.SetManagerFunc(guiStandardLayout)

	if err := gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, guiClose); err != nil {
		log.Panicln(err)
	}

	guiReady <- gui

	if err := gui.MainLoop(); err != nil {
		if err != gocui.ErrQuit {
			log.Panicln(err)
		}
	}
}

func guiDequeueOps() {
	for true {
		guiOpsMutex.Lock()
		guiOp := guiOps.Front()
		guiOpsMutex.Unlock()
		if guiOp != nil {
			singleton.opHandle(guiOp.Value.(Operation))
			guiOpsMutex.Lock()
			guiOps.Remove(guiOp)
			guiOpsMutex.Unlock()
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
	singleton.Closing <- true
	gui.DeleteKeybinding("", gocui.KeyCtrlC, gocui.ModNone)
	return gocui.ErrQuit
}
