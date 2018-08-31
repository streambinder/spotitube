package gui

import (
	"fmt"
	"log"
	"strings"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

func condPanelSelector(options uint64) uint64 {
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

func condParagraphStyle(message string, options uint64, width int) string {
	if (options & ParagraphStyleStandard) != 0 {
		message = messageParagraphStyle(message, ParagraphStyleStandard, width)
	} else if (options & ParagraphStyleAutoReturn) != 0 {
		message = messageParagraphStyle(message, ParagraphStyleAutoReturn, width)
	}
	return message
}

func condFontStyle(message string, options uint64) string {
	if (options & FontStyleBold) != 0 {
		message = MessageStyle(message, FontStyleBold)
	} else if (options & FontStyleUnderline) != 0 {
		message = MessageStyle(message, FontStyleUnderline)
	} else if (options & FontStyleReverse) != 0 {
		message = MessageStyle(message, FontStyleReverse)
	}
	return message
}

func condOrientationStyle(message string, options uint64, view *gocui.View) string {
	if (options & OrientationLeft) != 0 {
		message = messageOrientate(message, view, OrientationLeft)
	} else if (options & OrientationCenter) != 0 {
		message = messageOrientate(message, view, OrientationCenter)
	} else if (options & OrientationRight) != 0 {
		message = messageOrientate(message, view, OrientationRight)
	}
	return message
}

func condColorStyle(message string, options uint64) string {
	if (options & FontColorBlack) != 0 {
		message = messageColor(message, FontColorBlack)
	} else if (options & FontColorRed) != 0 {
		message = messageColor(message, FontColorRed)
	} else if (options & FontColorGreen) != 0 {
		message = messageColor(message, FontColorGreen)
	} else if (options & FontColorYellow) != 0 {
		message = messageColor(message, FontColorYellow)
	} else if (options & FontColorBlue) != 0 {
		message = messageColor(message, FontColorBlue)
	} else if (options & FontColorMagenta) != 0 {
		message = messageColor(message, FontColorMagenta)
	} else if (options & FontColorCyan) != 0 {
		message = messageColor(message, FontColorCyan)
	} else if (options & FontColorWhite) != 0 {
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

func messageColor(message string, colorConst int) string {
	colorFunc := color.New(FontColors[colorConst])
	return colorFunc.Sprintf(message)
}

func messageParagraphStyle(message string, styleConst int, width int) string {
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
